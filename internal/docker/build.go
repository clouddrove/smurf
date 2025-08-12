package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

// Color functions
func green(msg string) string  { return "\033[32m" + msg + "\033[0m" }
func red(msg string) string    { return "\033[31m" + msg + "\033[0m" }
func cyan(msg string) string   { return "\033[36m" + msg + "\033[0m" }
func yellow(msg string) string { return "\033[33m" + msg + "\033[0m" }

// logStep prints step info
func logStep(num, total int, msg string) {
	fmt.Printf("%s %s\n", cyan(fmt.Sprintf("[%d/%d]", num, total)), msg)
}
func logSuccess(msg string) {
	fmt.Println(green("✅ " + msg))
}
func logError(msg string) {
	fmt.Println(red("❌ " + msg))
}
func logInfo(msg string) {
	fmt.Println(cyan("ℹ " + msg))
}

// Build builds a Docker image
func Build(imageName, tag string, opts BuildOptions) error {
	totalSteps := 5
	step := 1

	logStep(step, totalSteps, "Initializing build...")
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		logError(fmt.Sprintf("docker client init failed: %v", err))
		return err
	}
	defer cli.Close()
	logSuccess("Docker client initialized")
	step++

	logStep(step, totalSteps, "Creating build context...")
	buildCtx, err := createTarball(opts.ContextDir, []string{".git", "node_modules"})
	if err != nil {
		logError(fmt.Sprintf("context creation failed: %v", err))
		return err
	}
	defer buildCtx.Close()
	logSuccess("Build context created")
	step++

	relDockerfilePath, err := filepath.Rel(opts.ContextDir, opts.DockerfilePath)
	if err != nil {
		logError(fmt.Sprintf("invalid dockerfile path: %v", err))
		return err
	}

	buildArgsPtr := make(map[string]*string)
	for k, v := range opts.BuildArgs {
		value := v
		buildArgsPtr[k] = &value
	}

	platform := opts.Platform
	if platform != "" {
		parts := strings.Split(platform, "/")
		if len(parts) != 2 {
			logError(fmt.Sprintf("invalid platform format. Expected os/arch, got: %s", opts.Platform))
			return fmt.Errorf("invalid platform format. Expected os/arch, got: %s", opts.Platform)
		}
	}

	fullImageName := fmt.Sprintf("%s:%s", imageName, tag)
	buildOptions := types.ImageBuildOptions{
		Tags:        []string{fullImageName},
		Dockerfile:  relDockerfilePath,
		NoCache:     opts.NoCache,
		Remove:      true,
		BuildArgs:   buildArgsPtr,
		Target:      opts.Target,
		Platform:    platform,
		Version:     types.BuilderV1,
		BuildID:     fmt.Sprintf("build-%d", time.Now().Unix()),
		PullParent:  true,
		NetworkMode: "default",
	}

	if opts.BuildKit {
		os.Setenv("DOCKER_BUILDKIT", "1")
		buildOptions.Version = types.BuilderBuildKit
	}

	logStep(step, totalSteps, "Running Docker build...")
	if opts.BuildKit {
		args := []string{"build", "--tag", fullImageName}
		if opts.NoCache {
			args = append(args, "--no-cache")
		}
		if relDockerfilePath != "" {
			args = append(args, "--file", relDockerfilePath)
		}
		for k, v := range opts.BuildArgs {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
		}
		if opts.Target != "" {
			args = append(args, "--target", opts.Target)
		}
		if platform != "" {
			args = append(args, "--platform", platform)
		}
		args = append(args, opts.ContextDir)

		cmd := exec.Command("docker", args...)
		cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")

		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			logError(fmt.Sprintf("failed to start build: %v", err))
			return err
		}

		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}()

		if err := cmd.Wait(); err != nil {
			logError(fmt.Sprintf("BuildKit build failed: %v", err))
			return err
		}

		logSuccess("Docker build completed successfully")
	} else {
		resp, err := cli.ImageBuild(ctx, buildCtx, buildOptions)
		if err != nil {
			logError(fmt.Sprintf("build failed: %v", err))
			return err
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var msg jsonmessage.JSONMessage
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if msg.Error != nil {
				logError(fmt.Sprintf("build error: %v", msg.Error))
				return err
			}
			if msg.Stream != "" {
				fmt.Print(msg.Stream)
			}
		}

		logSuccess("Docker build completed successfully")
	}
	step++

	logStep(step, totalSteps, "Inspecting image...")
	inspect, err := cli.ImageInspect(ctx, fullImageName)
	if err != nil {
		logError(fmt.Sprintf("failed to get image info: %v", err))
		return err
	}

	fmt.Printf("\n%s\n", green("=== Build Summary ==="))
	fmt.Printf("%s %s\n", cyan("Image:"), fullImageName)
	fmt.Printf("%s %s\n", cyan("ID:"), inspect.ID[:12])
	fmt.Printf("%s %.2f MB\n", cyan("Size:"), float64(inspect.Size)/1024/1024)
	fmt.Printf("%s %s/%s\n", cyan("Platform:"), inspect.Os, inspect.Architecture)
	fmt.Printf("%s %s\n", cyan("Created:"), inspect.Created[:19])
	fmt.Printf("%s %d\n", cyan("Layers:"), len(inspect.RootFS.Layers))
	logSuccess("Image inspection complete")
	return nil
}

// createTarball remains unchanged
func createTarball(srcDir string, excludePatterns []string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer tw.Close()

		filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(srcDir, file)
			if err != nil {
				return err
			}

			for _, pattern := range excludePatterns {
				if strings.HasPrefix(relPath, pattern) {
					if fi.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if relPath == "." {
				return nil
			}

			hdr, err := tar.FileInfoHeader(fi, relPath)
			if err != nil {
				return err
			}
			hdr.Name = relPath

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if fi.Mode().IsRegular() {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()

				if _, err := io.Copy(tw, f); err != nil {
					return err
				}
			}

			return nil
		})
	}()

	return pr, nil
}
