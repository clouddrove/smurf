package docker

import (
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
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/fatih/color"
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
		return fmt.Errorf("%v", color.RedString("docker client init failed: %v", err))
	}
	defer cli.Close()

	buildCtx, err := archive.TarWithOptions(opts.ContextDir, &archive.TarOptions{
		ExcludePatterns: []string{".git", "node_modules"},
	})
	if err != nil {
		spinner.Fail()
		return fmt.Errorf("%v", color.RedString("context creation failed: %v", err))
	}
	defer buildCtx.Close()

	relDockerfilePath, err := filepath.Rel(opts.ContextDir, opts.DockerfilePath)
	if err != nil {
		return fmt.Errorf("%v", color.RedString("invalid dockerfile path: %v", err))
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
			return fmt.Errorf("%v", color.RedString("invalid platform format. Expected os/arch, got: %s", platform))
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
		spinner.UpdateText("Running build with BuildKit enabled...")

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
			return fmt.Errorf("%v", color.RedString("failed to create stdout pipe: %v", err))
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("%v", color.RedString("failed to create stderr pipe: %v", err))
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("%v", color.RedString("failed to start build: %v", err))
		}

		spinner.Success()
		progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Building").Start()

		// Process stdout
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.HasPrefix(line, "Step ") {
					pterm.Info.Println(color.CyanString(line))
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
				pterm.Debug.Println(color.YellowString(line))
			}
		}()

		// Wait for the command to complete
		if err := cmd.Wait(); err != nil {
			progressBar.Stop()
			return fmt.Errorf("%v", color.RedString("BuildKit build failed: %v", err))
		}

		progressBar.Add(100 - progressBar.Current)

		time.Sleep(2 * time.Second)

		inspect, _, err := cli.ImageInspectWithRaw(ctx, fullImageName)
		if err != nil {
			return fmt.Errorf("%v", color.RedString("failed to get image info: %v", err))
		}

		panel := pterm.DefaultBox.WithTitle("Build Complete").Sprintf(` %s %s %s %s %s %s %s `,
			color.GreenString("✓ Image Built Successfully"),
			color.CyanString("Image: %s", fullImageName),
			color.CyanString("ID: %s", inspect.ID[:12]),
			color.CyanString("Size: %.2f MB", float64(inspect.Size)/1024/1024),
			color.CyanString("Platform: %s/%s", inspect.Os, inspect.Architecture),
			color.CyanString("Created: %s", inspect.Created[:19]),
			color.CyanString("Layers: %d", len(inspect.RootFS.Layers)),
		)

		fmt.Println(panel)
		return nil
	}

	// The rest of your original Build function continues below for non-BuildKit builds
	resp, err := cli.ImageBuild(ctx, buildCtx, buildOptions)
	if err != nil {
		spinner.Fail()
		return fmt.Errorf("%v", color.RedString("build failed: %v", err))
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
			return fmt.Errorf("%v", color.RedString("build error: %v", msg.Error))
		}

		if msg.Stream != "" {
			if strings.HasPrefix(msg.Stream, "Step ") {
				pterm.Info.Println(color.CyanString(msg.Stream))
				progressBar.Add(5)
			} else {
				trimmed := strings.TrimSpace(msg.Stream)
				if trimmed != "" {
					pterm.Debug.Println(trimmed)
				}
			}
		}

		if msg.Status != "" {
			pterm.Debug.Println(color.YellowString(msg.Status))
			progressBar.Add(1)
		}
	}

	if lastError != nil {
		progressBar.Stop()
		return fmt.Errorf("%v", color.RedString("build process error: %v", lastError))
	}

	progressBar.Add(100 - progressBar.Current)

	time.Sleep(2 * time.Second)

	inspect, _, err := cli.ImageInspectWithRaw(ctx, fullImageName)
	if err != nil {
		return fmt.Errorf("%v", color.RedString("failed to get image info: %v", err))
	}

	panel := pterm.DefaultBox.WithTitle("Build Complete").Sprintf(` %s %s %s %s %s %s %s `,
		color.GreenString("✓ Image Built Successfully"),
		color.CyanString("Image: %s", fullImageName),
		color.CyanString("ID: %s", inspect.ID[:12]),
		color.CyanString("Size: %.2f MB", float64(inspect.Size)/1024/1024),
		color.CyanString("Platform: %s/%s", inspect.Os, inspect.Architecture),
		color.CyanString("Created: %s", inspect.Created[:19]),
		color.CyanString("Layers: %d", len(inspect.RootFS.Layers)),
	)

	fmt.Println(panel)
	return nil
}
