package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// TagImage tags a Docker image with the specified source and target tags.
// It displays a spinner with progress updates and prints a success message upon completion.
func TagImage(opts TagOptions) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return logAndReturnError("Error creating Docker client : %v", err)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Tagging image %s as %s...", opts.Source, opts.Target))
	if err := cli.ImageTag(ctx, opts.Source, opts.Target); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to tag image: %v", err))
		return logAndReturnError("Failed to tag image : %v", err)
	}

	spinner.Success(fmt.Sprintf("Successfully tagged %s as %s", opts.Source, opts.Target))
	color.New(color.FgGreen).Printf("Image tagged successfully: %s -> %s\n", opts.Source, opts.Target)
	return nil
}
