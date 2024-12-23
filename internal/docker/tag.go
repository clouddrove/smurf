package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
)

// TagImage tags a local Docker image for use in a remote repository.
func TagImage(opts TagOptions) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		color.New(color.FgRed).Printf("Error creating Docker client: %v\n", err)
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Tagging image %s as %s...", opts.Source, opts.Target))
	if err := cli.ImageTag(ctx, opts.Source, opts.Target); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to tag image: %v", err))
		color.New(color.FgRed).Println(err)
		return err
	}

	spinner.Success(fmt.Sprintf("Successfully tagged %s as %s", opts.Source, opts.Target))
	color.New(color.FgGreen).Printf("Image tagged successfully: %s -> %s\n", opts.Source, opts.Target)
	return nil
}
