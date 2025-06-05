package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/pterm/pterm"
)

// TagImage tags a Docker image with the specified source and target tags.
// It displays a spinner with progress updates and prints a success message upon completion.
func TagImage(opts TagOptions) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		pterm.Error.Printf("Error creating Docker client : %v", err)
		return fmt.Errorf("error creating Docker client : %v", err)
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Tagging image %s as %s...", opts.Source, opts.Target))
	if err := cli.ImageTag(ctx, opts.Source, opts.Target); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to tag image: %v", err))
		return fmt.Errorf("failed to tag image : %v", err)
	}

	spinner.Success(fmt.Sprintf("Successfully tagged %s as %s\n", opts.Source, opts.Target))
	return nil
}
