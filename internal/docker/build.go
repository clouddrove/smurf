package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/api/types"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/moby/term"
)

// Build builds a Docker image using the provided options.
func Build(imageName, tag string, opts BuildOptions) error {
	if opts.Platform == "" && isM1Mac() {
		opts.Platform = "linux/amd64"
	}

	if err := validateBuildContext(opts.ContextDir); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(opts.Timeout),
	)
	if err != nil {
		return fmt.Errorf("docker client creation failed: %w", err)
	}
	defer cli.Close()

	tarStream, err := createOptimizedTarArchive(opts.ContextDir)
	if err != nil {
		return fmt.Errorf("tar archive error: %w", err)
	}

	buildOptions := types.ImageBuildOptions{
		Tags:        []string{fmt.Sprintf("%s:%s", imageName, tag)},
		Dockerfile:  filepath.Base(opts.DockerfilePath),
		NoCache:     opts.NoCache,
		BuildArgs:   convertToInterfaceMap(opts.BuildArgs),
		Target:      opts.Target,
		Remove:      true,
		ForceRemove: true,
		PullParent:  false,
		Platform:    opts.Platform,
	}

	spinner, _ := pterm.DefaultSpinner.Start("Building Docker image...")

	buildResponse, err := cli.ImageBuild(ctx, tarStream, buildOptions)
	if err != nil {
		spinner.Fail("Build initialization failed")
		return fmt.Errorf("image build error: %w", err)
	}
	defer buildResponse.Body.Close()

	outFd := os.Stdout.Fd()
	_ = term.IsTerminal(outFd)

	dec := json.NewDecoder(buildResponse.Body)
BuildLoop:
	for {
		var msg jsonmessage.JSONMessage
		if err := dec.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			spinner.Fail("Build logging encountered errors")
			return fmt.Errorf("error decoding build output: %w", err)
		}

		if msg.Error != nil {
			spinner.Fail("Build failed")
			color.Red("Error: %v\n", msg.Error)
			return fmt.Errorf("docker build error: %v", msg.Error)
		}

		if msg.Stream != "" {
			line := strings.TrimSpace(msg.Stream)
			if strings.HasPrefix(line, "Step ") ||
				strings.HasPrefix(line, "Successfully built") ||
				strings.HasPrefix(line, "Successfully tagged") ||
				strings.HasPrefix(line, "--->") ||
				strings.Contains(line, "Using cache") {

				if strings.HasPrefix(line, "Step ") {
					color.Cyan(line + "\n")
				} else if strings.HasPrefix(line, "Successfully") {
					color.Green(line + "\n")
				} else {
					fmt.Println(line)
				}
				if strings.HasPrefix(line, "Successfully built") {
					spinner.Success("Docker image built successfully")
					color.Green("Successfully built %s:%s\n", imageName, tag)
					break BuildLoop
				}
			}
		}

	}

	return nil
}
