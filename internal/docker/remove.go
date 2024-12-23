package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/pterm/pterm"
)

// RemoveImage removes a Docker image based on the provided flags.
func RemoveImage(imageTag string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	pterm.Info.Println("Removing local Docker image:", imageTag)
	spinner, _ := pterm.DefaultSpinner.Start("Removing image...")

	_, err = cli.ImageRemove(ctx, imageTag, image.RemoveOptions{Force: true})
	if err != nil {
		spinner.Fail("Failed to remove local Docker image:", imageTag)
		return fmt.Errorf("failed to remove local Docker image: %w", err)
	}

	spinner.Success("Successfully removed local Docker image:", imageTag)
	return nil
}
