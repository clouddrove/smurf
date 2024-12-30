package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/pterm/pterm"
)

// PushImage pushes the specified Docker image to the Docker Hub.
// It authenticates with Docker Hub, tags the image, and pushes it to the registry.
// It displays a spinner with progress updates and prints the push response messages.
func PushImage(opts PushOptions) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		pterm.Error.Println("Error creating Docker client:", err)
		return err
	}

	authConfig := registry.AuthConfig{
		Username: os.Getenv("DOCKER_USERNAME"),
		Password: os.Getenv("DOCKER_PASSWORD"),
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		pterm.Error.Println("Error encoding auth config:", err)
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Pushing image %s...", opts.ImageName))
	options := image.PushOptions{
		RegistryAuth: authStr,
	}

	responseBody, err := cli.ImagePush(ctx, opts.ImageName, options)
	if err != nil {
		spinner.Fail("Failed to push the image: " + err.Error())
		return err
	}
	defer responseBody.Close()

	return handleDockerResponse(responseBody, spinner, opts)
}
