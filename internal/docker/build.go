package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		spinner.Fail()
		return fmt.Errorf(color.RedString("docker client init failed: %v", err))
	}
	defer cli.Close()

	buildCtx, err := archive.TarWithOptions(opts.ContextDir, &archive.TarOptions{
		ExcludePatterns: []string{".git", "node_modules"},
	})
	if err != nil {
		spinner.Fail()
		return fmt.Errorf(color.RedString("context creation failed: %v", err))
	}
	defer buildCtx.Close()

	relDockerfilePath, err := filepath.Rel(opts.ContextDir, opts.DockerfilePath)
	if err != nil {
		return fmt.Errorf(color.RedString("invalid dockerfile path: %v", err))
	}

	buildArgsPtr := make(map[string]*string)
	for k, v := range opts.BuildArgs {
		value := v
		buildArgsPtr[k] = &value
	}

	fullImageName := fmt.Sprintf("%s:%s", imageName, tag)
	buildOptions := types.ImageBuildOptions{
		Tags:        []string{fullImageName},
		Dockerfile:  relDockerfilePath,
		NoCache:     opts.NoCache,
		Remove:      true,
		BuildArgs:   buildArgsPtr,
		Target:      opts.Target,
		Platform:    opts.Platform,
		PullParent:  false,
		NetworkMode: "default",
		BuildID:     fmt.Sprintf("build-%d", time.Now().Unix()),
	}

	resp, err := cli.ImageBuild(ctx, buildCtx, buildOptions)
	if err != nil {
		return fmt.Errorf(color.RedString("build failed: %v", err))
	}
	defer resp.Body.Close()

	spinner.Success()
	progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Building").Start()

	var imageID string
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
			progressBar.Stop()
			return msg.Error
		}

		if msg.Stream != "" && strings.HasPrefix(msg.Stream, "Step ") {
			pterm.Info.Println(color.CyanString(msg.Stream))
			progressBar.Add(5)
		}

		if msg.Aux != nil {
			var result struct{ ID string }
			if err := json.Unmarshal(*msg.Aux, &result); err == nil {
				imageID = result.ID
			}
		}
	}

	progressBar.Add(100 - progressBar.Current)

	inspect, _, err := cli.ImageInspectWithRaw(ctx, fullImageName)
	if err != nil {
		return fmt.Errorf(color.RedString("failed to get image info: %v", err))
	}

	panel := pterm.DefaultBox.WithTitle("Build Complete").Sprintf(`
%s
%s
%s
%s
%s
%s
%s
`,
		color.GreenString("✓ Image Built Successfully"),
		color.CyanString("Image: %s", fullImageName),
		color.CyanString("ID: %s", imageID[:12]),
		color.CyanString("Size: %.2f MB", float64(inspect.Size)/1024/1024),
		color.CyanString("Platform: %s/%s", inspect.Os, inspect.Architecture),
		color.CyanString("Created: %s", inspect.Created[:19]),
		color.CyanString("Layers: %d", len(inspect.RootFS.Layers)),
	)

	fmt.Println(panel)
	return nil
}
