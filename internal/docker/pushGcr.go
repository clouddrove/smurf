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

func PushImageToGCR(projectID, imageName string) error {
	ctx := context.Background()

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
	if strings.Contains(imageName, "gcr.io") {
		registryHost = "gcr.io"
	} else {
		registryHost = fmt.Sprintf("gcr.io/%s", projectID)
	}
	taggedImage := fmt.Sprintf("%s/%s", registryHost, imageName)
	err = dockerClient.ImageTag(ctx, imageName, taggedImage)
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

	link := fmt.Sprintf("https://console.cloud.google.com/gcr/images/%s/%s?project=%s", projectID, imageName, projectID)
	color.New(color.FgGreen).Printf("Image pushed to GCR: %s\n", link)
	color.New(color.FgGreen).Printf("Successfully pushed image '%s' to GCR\n", taggedImage)
	return nil
}
