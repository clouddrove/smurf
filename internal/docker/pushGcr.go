package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"golang.org/x/oauth2/google"
)

// / PushImageToGCR pushes the specified Docker image to the specified Google Container Registry.
// It authenticates with Google Cloud, retrieves the registry details and credentials, tags the image,
// and pushes it to the registry. It displays a spinner with progress updates and prints the
// push response messages. Upon successful completion, it prints a success message with a link
// to the pushed image in the GCR.
func PushImageToGCR(projectID, imageNameWithTag string) error {
	ctx := context.Background()

	// Parse image name and tag
	imageName := imageNameWithTag
	imageTag := "latest"
	if parts := strings.Split(imageNameWithTag, ":"); len(parts) == 2 {
		imageName = parts[0]
		imageTag = parts[1]
	}

	spinner, _ := pterm.DefaultSpinner.Start("Authenticating with Google Cloud...")
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		spinner.Fail("Failed to authenticate with Google Cloud")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Authenticated with Google Cloud")

	spinner, _ = pterm.DefaultSpinner.Start("Obtaining access token...")
	tokenSource := creds.TokenSource
	token, err := tokenSource.Token()
	if err != nil {
		spinner.Fail("Failed to obtain access token")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Access token obtained")

	spinner, _ = pterm.DefaultSpinner.Start("Creating Docker client...")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		spinner.Fail("Failed to create Docker client")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Docker client created")

	spinner, _ = pterm.DefaultSpinner.Start("Tagging the image...")
	var registryHost string

	// Check if the image name already includes a registry
	if strings.Contains(imageName, "gcr.io") || strings.Contains(imageName, "docker.pkg.dev") {
		// Use the image name as is, since it already includes a registry
		registryHost = ""
	} else {
		registryHost = fmt.Sprintf("gcr.io/%s/", projectID)
	}

	// Construct the proper tagged image with registry and tag
	sourceImage := fmt.Sprintf("%s:%s", imageName, imageTag)
	taggedImage := fmt.Sprintf("%s%s:%s", registryHost, imageName, imageTag)

	// If the image already has a full registry path, don't prepend registryHost
	if registryHost == "" {
		taggedImage = sourceImage
	}

	err = dockerClient.ImageTag(ctx, sourceImage, taggedImage)
	if err != nil {
		spinner.Fail("Failed to tag the image")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	spinner.Success("Image tagged")

	spinner, _ = pterm.DefaultSpinner.Start("Pushing the image to GCR...")
	authConfig := registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: "https://gcr.io",
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		spinner.Fail("Failed to encode authentication credentials")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}

	pushOptions := image.PushOptions{
		RegistryAuth: encodedAuth,
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, pushOptions)
	if err != nil {
		spinner.Fail("Failed to push the image")
		color.New(color.FgRed).Printf("Error: %v\n", err)
		return err
	}
	defer pushResponse.Close()

	dec := json.NewDecoder(pushResponse)
	progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("Pushing to GCR").Start()
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			spinner.Fail("Failed to read push response")
			color.New(color.FgRed).Printf("Error: %v\n", err)
			return err
		}
		if event.Error != nil {
			spinner.Fail("Failed to push the image")
			color.New(color.FgRed).Printf("Error: %v\n", event.Error)
			return event.Error
		}
		if event.Progress != nil && event.Progress.Total > 0 {
			progress := int(float64(event.Progress.Current) / float64(event.Progress.Total) * 100)
			if progress > 100 {
				progress = 100
			}
			progressBar.Add(progress - progressBar.Current)
		}
	}
	progressBar.Stop()
	spinner.Success("Image pushed to GCR")

	// Construct the proper console link
	registryType := "gcr"
	if strings.Contains(imageName, "docker.pkg.dev") {
		registryType = "artifacts/repository"
	}

	link := fmt.Sprintf("https://console.cloud.google.com/%s/images/%s/%s?project=%s",
		registryType, projectID, imageName, projectID)

	color.New(color.FgGreen).Printf("Image pushed to GCR: %s\n", link)
	color.New(color.FgGreen).Printf("Successfully pushed image '%s' to GCR\n", taggedImage)
	return nil
}
