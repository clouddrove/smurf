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

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
)

// Color functions
func green(msg string) string   { return "\033[32m" + msg + "\033[0m" }
func red(msg string) string     { return "\033[31m" + msg + "\033[0m" }
func cyan(msg string) string    { return "\033[36m" + msg + "\033[0m" }
func blue(msg string) string    { return "\033[34m" + msg + "\033[0m" }
func magenta(msg string) string { return "\033[35m" + msg + "\033[0m" }
func bold(msg string) string    { return "\033[1m" + msg + "\033[0m" }

// Step tracker with enhanced error coloring
type stepTracker struct {
	current int
	total   int
	start   time.Time
}

func newStepTracker(total int) *stepTracker {
	return &stepTracker{
		total: total,
		start: time.Now(),
	}
}

func (st *stepTracker) logStep(msg string) {
	st.current++
	fmt.Printf("\n%s %s\n", cyan(fmt.Sprintf("STEP %d/%d:", st.current, st.total)), bold(msg))
}

func (st *stepTracker) completeStep(success bool, msg string) {
	elapsed := time.Since(st.start).Round(time.Millisecond)

	if success {
		fmt.Printf("%s %s (%s)\n", green("✓"), msg, cyan(elapsed.String()))
	} else {
		// Make entire failed step red including timestamp
		fullMsg := fmt.Sprintf("✗ %s (%s)", msg, elapsed.String())
		fmt.Printf("%s\n", red(fullMsg))
	}
	st.start = time.Now()
}

func printDivider() {
	fmt.Println(green("⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯"))
}

func printBuildSummary(inspect types.ImageInspect, fullImageName string) {
	printDivider()
	fmt.Printf("%s %s\n\n", green(bold("✓ BUILD SUCCESS")), magenta(fmt.Sprintf("%s [%s]", fullImageName, inspect.ID[:12])))

	fmt.Printf("%s %s\n", blue("▸ Platform:"), fmt.Sprintf("%s/%s", inspect.Os, inspect.Architecture))

	createdTime, err := time.Parse(time.RFC3339Nano, inspect.Created)
	if err == nil {
		fmt.Printf("%s %s\n", blue("▸ Created:"), createdTime.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("%s %s\n", blue("▸ Created:"), inspect.Created)
	}

	fmt.Printf("%s %.2f MB\n", blue("▸ Size:"), float64(inspect.Size)/1024/1024)
	fmt.Printf("%s %d layers\n", blue("▸ Layers:"), len(inspect.RootFS.Layers))

	if len(inspect.Config.Labels) > 0 {
		fmt.Printf("\n%s\n", blue("Labels:"))
		for k, v := range inspect.Config.Labels {
			fmt.Printf("  %s: %s\n", cyan(k), v)
		}
	}

	printDivider()
}

func Build(imageName, tag string, opts BuildOptions, useAI bool) error {
	tracker := newStepTracker(3)

	tracker.logStep("Initializing build...")
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		tracker.completeStep(false, fmt.Sprintf("Docker client init failed: %v", err))
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("%v", err.Error())
	}
	defer cli.Close()

	version, err := cli.ServerVersion(ctx)
	if err != nil {
		tracker.completeStep(false, "Could not get Docker version info")
	} else {
		tracker.completeStep(true, fmt.Sprintf("Docker client initialized [version: %s, API: %s]", version.Version, version.APIVersion))
	}

	tracker.logStep("Creating build context...")
	if len(opts.Excludes) > 0 {
		fmt.Printf("%s Excluding: %s\n", blue("ℹ"), strings.Join(opts.Excludes, ", "))
	}

	buildCtx, err := createTarball(opts.ContextDir, opts.Excludes)
	if err != nil {
		tracker.completeStep(false, fmt.Sprintf("Context creation failed: %v", err))
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("%v", err.Error())
	}
	defer buildCtx.Close()

	// Get context size
	tmpFile, err := os.CreateTemp("", "docker-build-context-")
	if err == nil {
		defer os.Remove(tmpFile.Name())
		size, err := io.Copy(tmpFile, buildCtx)
		if err == nil {
			tracker.completeStep(true, fmt.Sprintf("Build context created [%.1f MB]", float64(size)/1024/1024))
			buildCtx, err = createTarball(opts.ContextDir, opts.Excludes)
			if err != nil {
				tracker.completeStep(false, fmt.Sprintf("Failed to recreate build context: %v", err))
				ai.AIExplainError(useAI, err.Error())
				return fmt.Errorf("%v", err.Error())
			}
			defer buildCtx.Close()
		}
	} else {
		tracker.completeStep(true, "Build context created")
	}

	relDockerfilePath, err := filepath.Rel(opts.ContextDir, opts.DockerfilePath)
	if err != nil {
		tracker.completeStep(false, fmt.Sprintf("Invalid Dockerfile path: %v", err))
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("%v", err.Error())
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
			errMsg := fmt.Sprintf("invalid platform format. Expected os/arch, got: %s", opts.Platform)
			tracker.completeStep(false, errMsg)
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("%v", errMsg)
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
		Labels:      opts.Labels,
	}

	if opts.BuildKit {
		os.Setenv("DOCKER_BUILDKIT", "1")
		buildOptions.Version = types.BuilderBuildKit
	}

	tracker.logStep("Running Docker build...")
	if opts.BuildKit {
		args := []string{"build", "--progress=plain", "--tag", fullImageName}
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
			tracker.completeStep(false, fmt.Sprintf("Failed to start build: %v", err))
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("%v", err.Error())
		}

		// Stream output with error coloring
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.Contains(strings.ToLower(line), "error") {
					fmt.Printf("%s\n", red(line))
				} else if strings.HasPrefix(line, "=>") {
					fmt.Printf("%s %s\n", cyan("→"), line)
				} else {
					fmt.Println(line)
				}
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				fmt.Printf("%s\n", red(scanner.Text()))
			}
		}()

		if err := cmd.Wait(); err != nil {
			tracker.completeStep(false, fmt.Sprintf("BuildKit build failed: %v", err))
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("%v", err.Error())
		}

		tracker.completeStep(true, "Docker build completed successfully")
	} else {
		resp, err := cli.ImageBuild(ctx, buildCtx, buildOptions)
		if err != nil {
			tracker.completeStep(false, fmt.Sprintf("Build failed: %v", err))
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("%v", err)
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var msg jsonmessage.JSONMessage
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					break
				}
				tracker.completeStep(false, fmt.Sprintf("Build error: %v", err))
				ai.AIExplainError(useAI, err.Error())
				return fmt.Errorf("%v", err)
			}
			if msg.Error != nil {
				tracker.completeStep(false, fmt.Sprintf("Build error: %v", msg.Error))
				errMsg := fmt.Sprint(msg.Error)
				ai.AIExplainError(useAI, errMsg)
				return fmt.Errorf("%v", msg.Error)
			}
			if msg.Stream != "" {
				if strings.Contains(strings.ToLower(msg.Stream), "error") {
					fmt.Printf("%s", red(msg.Stream))
				} else {
					fmt.Print(msg.Stream)
				}
			}
		}

		tracker.completeStep(true, "Docker build completed successfully")
	}

	// tracker.logStep("Inspecting image...")
	inspect, _, err := cli.ImageInspectWithRaw(ctx, fullImageName)
	if err != nil {
		tracker.completeStep(false, fmt.Sprintf("Failed to inspect image: %v", err))
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("%v", err.Error())
	}

	// tracker.completeStep(true, "Image inspection complete")
	printBuildSummary(inspect, fullImageName)
	return nil
}

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
				if matched, _ := filepath.Match(pattern, relPath); matched {
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
