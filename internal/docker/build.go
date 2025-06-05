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
	"github.com/pterm/pterm"
)

// Build creates and tags a Docker image using the specified build context and options.
// It displays a spinner and progress bar for user feedback, and inspects the final image
// to provide details like size, platform, and creation time upon successful completion.
func Build(imageName, tag string, opts BuildOptions) error {
	spinner, _ := pterm.DefaultSpinner.Start("Initializing build...")

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		spinner.Fail()
		pterm.Error.Println("docker client init failed: ", err)
		return fmt.Errorf("docker client init failed: %v", err)
	}
	defer cli.Close()

	buildCtx, err := createTarball(opts.ContextDir, []string{".git", "node_modules"})
	if err != nil {
		pterm.Error.Println("context creation failed: ", err)
		return fmt.Errorf("context creation failed: %v", err)
	}
	defer buildCtx.Close()

	// Get relative path to Dockerfile within context
	relDockerfilePath, err := filepath.Rel(opts.ContextDir, opts.DockerfilePath)
	if err != nil {
		pterm.Error.Println("invalid dockerfile path: ", err)
		return fmt.Errorf("invalid dockerfile path: %v", err)
	}

	buildArgsPtr := make(map[string]*string)
	for k, v := range opts.BuildArgs {
		value := v
		buildArgsPtr[k] = &value
	}

	// Validate platform format if specified
	platform := opts.Platform
	if platform != "" {
		parts := strings.Split(platform, "/")
		if len(parts) != 2 {
			pterm.Error.Println("invalid platform format. Expected os/arch, got: ", opts.Platform)
			return fmt.Errorf("invalid platform format. Expected os/arch, got: %s", opts.Platform)
		}
	}

	fullImageName := fmt.Sprintf("%s:%s", imageName, tag)
	// In the Build function, modify the buildOptions
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

	// Add this before the ImageBuild call
	if opts.BuildKit {
		// Enable BuildKit
		os.Setenv("DOCKER_BUILDKIT", "1")
	}

	if opts.BuildKit {
		buildOptions.Version = types.BuilderBuildKit
	}

	// Build function modification

	// Modify this part in your docker.Build function
	if opts.BuildKit {
		spinner.UpdateText("Running build with BuildKit enabled...\n")

		// Construct docker command arguments
		args := []string{"build"}

		// Add the tag
		args = append(args, "--tag", fullImageName)

		// Add other build options
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

		// Add the context directory as the last argument
		args = append(args, opts.ContextDir)

		// Create and configure the command
		cmd := exec.Command("docker", args...)

		// Set environment with BuildKit enabled
		cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")

		// Create pipes for stdout and stderr
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			pterm.Error.Println("failed to create stdout pipe: ", err)
			return fmt.Errorf("failed to create stdout pipe: %v", err)
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			pterm.Error.Println("failed to create stderr pipe: ", err)
			return fmt.Errorf("failed to create stderr pipe: %v", err)
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			pterm.Error.Println("failed to start build: ", err)
			return fmt.Errorf("failed to start build: %v", err)
		}

		spinner.Success()
		progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Building").Start()

		// Process stdout
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Step ") {
					pterm.Info.Println(line)
					progressBar.Add(5)
				} else {
					trimmed := strings.TrimSpace(line)
					if trimmed != "" {
						pterm.Debug.Println(trimmed)
					}
				}
			}
		}()

		// Process stderr
		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				line := scanner.Text()
				pterm.Debug.Println(line)
			}
		}()

		// Wait for the command to complete
		if err := cmd.Wait(); err != nil {
			progressBar.Stop()
			pterm.Error.Println("BuildKit build failed: ", err)
			return fmt.Errorf("buildKit build failed: %v", err)
		}

		progressBar.Add(100 - progressBar.Current)

		time.Sleep(2 * time.Second)

		inspect, err := cli.ImageInspect(ctx, fullImageName)
		if err != nil {
			pterm.Error.Println("failed to get image info: ", err)
			return fmt.Errorf("failed to get image info: %v", err)
		}

		panel := pterm.DefaultBox.WithTitle("Build Complete").Sprintf(` %s %s %s %s %s %s %s `,
			pterm.FgCyan.Sprintf("✓ Image Built Successfully"),
			pterm.FgCyan.Sprintf("Image: %s", fullImageName),
			pterm.FgCyan.Sprintf("ID: %s", inspect.ID[:12]),
			pterm.FgCyan.Sprintf("Size: %.2f MB", float64(inspect.Size)/1024/1024),
			pterm.FgCyan.Sprintf("Platform: %s/%s", inspect.Os, inspect.Architecture),
			pterm.FgCyan.Sprintf("Created: %s", inspect.Created[:19]),
			pterm.FgCyan.Sprintf("Layers: %d", len(inspect.RootFS.Layers)),
		)

		fmt.Println(panel)
		return nil
	}

	// The rest of your original Build function continues below for non-BuildKit builds
	resp, err := cli.ImageBuild(ctx, buildCtx, buildOptions)
	if err != nil {
		spinner.Fail()
		pterm.Error.Println("build failed: ", err)
		return fmt.Errorf("build failed: %v", err)
	}
	defer resp.Body.Close()

	spinner.Success()
	progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Building").Start()

	var lastError error
	decoder := json.NewDecoder(resp.Body)
	for {
		var msg jsonmessage.JSONMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			lastError = err
			break
		}

		if msg.Error != nil {
			progressBar.Stop()
			pterm.Error.Println("build error: ", msg.Error)
			return fmt.Errorf("build error: %v", msg.Error)
		}

		if msg.Stream != "" {
			if strings.HasPrefix(msg.Stream, "Step ") {
				pterm.Info.Println(msg.Stream)
				progressBar.Add(5)
			} else {
				trimmed := strings.TrimSpace(msg.Stream)
				if trimmed != "" {
					pterm.Debug.Println(trimmed)
				}
			}
		}

		if msg.Status != "" {
			pterm.Debug.Println(msg.Status)
			progressBar.Add(1)
		}
	}

	if lastError != nil {
		progressBar.Stop()
		pterm.Error.Println("build process error: ", lastError)
		return fmt.Errorf("build process error: %v", lastError)
	}

	progressBar.Add(100 - progressBar.Current)

	time.Sleep(2 * time.Second)

	inspect, err := cli.ImageInspect(ctx, fullImageName)
	if err != nil {
		pterm.Error.Println("failed to get image info: ", err)
		return fmt.Errorf("failed to get image info: %v", err)
	}

	panel := pterm.DefaultBox.WithTitle("Build Complete").Sprintf(` %s %s %s %s %s %s %s `,
		pterm.FgCyan.Sprintf("✓ Image Built Successfully"),
		pterm.FgCyan.Sprintf("Image: %s", fullImageName),
		pterm.FgCyan.Sprintf("ID: %s", inspect.ID[:12]),
		pterm.FgCyan.Sprintf("Size: %.2f MB", float64(inspect.Size)/1024/1024),
		pterm.FgCyan.Sprintf("Platform: %s/%s", inspect.Os, inspect.Architecture),
		pterm.FgCyan.Sprintf("Created: %s", inspect.Created[:19]),
		pterm.FgCyan.Sprintf("Layers: %d", len(inspect.RootFS.Layers)),
	)

	fmt.Println(panel)
	return nil
}

// createTarball creates a tar archive of the specified directory, excluding specified patterns.
func createTarball(srcDir string, excludePatterns []string) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	tw := tar.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer tw.Close()

		filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				pterm.Error.Println("failed : ", err)
				return fmt.Errorf("failed : %v", err)
			}

			relPath, err := filepath.Rel(srcDir, file)
			if err != nil {
				pterm.Error.Println("failed to get relative path : ", err)
				return fmt.Errorf("failed to get relative path : %v", err)
			}

			// Skip excluded patterns
			for _, pattern := range excludePatterns {
				if strings.HasPrefix(relPath, pattern) {
					if fi.IsDir() {
						pterm.Error.Print(filepath.SkipDir)
						return filepath.SkipDir
					}
					return nil
				}
			}

			// Skip the source directory itself
			if relPath == "." {
				return nil
			}

			hdr, err := tar.FileInfoHeader(fi, relPath)
			if err != nil {
				pterm.Error.Println("failed to get file info header: ", err)
				return fmt.Errorf("failed to get file info header: %v", err)
			}
			hdr.Name = relPath

			if err := tw.WriteHeader(hdr); err != nil {
				pterm.Error.Println("failed to write header : ", err)
				return fmt.Errorf("failed to write header : %v", err)
			}

			if fi.Mode().IsRegular() {
				f, err := os.Open(file)
				if err != nil {
					pterm.Error.Println("failed to open file: ", err)
					return fmt.Errorf("failed to open file: %v", err)
				}
				defer f.Close()

				if _, err := io.Copy(tw, f); err != nil {
					pterm.Error.Println("failed: ", err)
					return fmt.Errorf("failed: %v", err)
				}
			}

			return nil
		})
	}()

	return pr, nil
}
