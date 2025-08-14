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
	"golang.org/x/oauth2/google"
)

// Color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

// PushImageToGCR pushes the specified Docker image to Google Container Registry
// with enhanced colorful logging that works in both terminals and GitHub Actions.
func PushImageToGCR(projectID, imageNameWithTag string) error {
	ctx := context.Background()

	// Parse image name and tag
	imageName := imageNameWithTag
	imageTag := "latest"
	if parts := strings.Split(imageNameWithTag, ":"); len(parts) == 2 {
		imageName = parts[0]
		imageTag = parts[1]
	}

	printStep("Authenticating with Google Cloud...")
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		printError("Failed to authenticate with Google Cloud")
		return fmt.Errorf("failed to authenticate with Google Cloud: %v", err)
	}
	printSuccess("Authenticated with Google Cloud")

	printStep("Obtaining access token...")
	tokenSource := creds.TokenSource
	token, err := tokenSource.Token()
	if err != nil {
		printError("Failed to obtain access token")
		return fmt.Errorf("failed to obtain access token: %v", err)
	}
	printSuccess("Access token obtained")

	printStep("Creating Docker client...")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		printError("Failed to create Docker client")
		return fmt.Errorf("failed to create Docker client: %v", err)
	}
	printSuccess("Docker client created")

	printStep("Tagging the image...")
	var registryHost string

	// Check if the image name already includes a registry
	if strings.Contains(imageName, "gcr.io") || strings.Contains(imageName, "docker.pkg.dev") {
		// Use the image name as is
		registryHost = ""
	} else {
		registryHost = fmt.Sprintf("gcr.io/%s/", projectID)
	}

	// Construct the proper tagged image
	sourceImage := fmt.Sprintf("%s:%s", imageName, imageTag)
	taggedImage := fmt.Sprintf("%s%s:%s", registryHost, imageName, imageTag)

	if registryHost == "" {
		taggedImage = sourceImage
	}

	err = dockerClient.ImageTag(ctx, sourceImage, taggedImage)
	if err != nil {
		printError("Failed to tag the image")
		return fmt.Errorf("failed to tag the image: %v", err)
	}
	printSuccess(fmt.Sprintf("Image tagged as: %s%s%s", colorCyan, taggedImage, colorReset))

	printStep("Pushing the image to GCR...")
	authConfig := registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: "https://gcr.io",
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		printError("Failed to encode authentication credentials")
		return fmt.Errorf("failed to encode authentication credentials: %v", err)
	}

	pushOptions := image.PushOptions{
		RegistryAuth: encodedAuth,
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, pushOptions)
	if err != nil {
		printError("Failed to push the image")
		return fmt.Errorf("failed to push the image: %v", err)
	}
	defer pushResponse.Close()

	// Enhanced push progress with colors
	var lastStatus string
	dec := json.NewDecoder(pushResponse)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			printError("Failed to read push response")
			return fmt.Errorf("failed to read push response: %v", err)
		}
		if event.Error != nil {
			printError("Failed to push the image")
			return fmt.Errorf("failed to push the image: %v", err)
		}
		if event.Status != "" && event.Status != lastStatus {
			printProgress(event.Status)
			lastStatus = event.Status
		}
	}

	// Construct the proper console link
	registryType := "gcr"
	if strings.Contains(imageName, "docker.pkg.dev") {
		registryType = "artifacts/repository"
	}

	link := fmt.Sprintf("https://console.cloud.google.com/%s/images/%s/%s?project=%s",
		registryType, projectID, imageName, projectID)

	printSuccess(fmt.Sprintf("Image successfully pushed to GCR: %s%s%s", colorCyan, link, colorReset))
	fmt.Printf("::set-output name=image_url::%s\n", link) // GitHub Actions output
	return nil
}

// Helper functions for colored output
func printStep(message string) {
	fmt.Printf("%s==>%s %s%s%s\n", colorBlue, colorReset, colorWhite, message, colorReset)
}

func printSuccess(message string) {
	fmt.Printf("%s✓%s %s%s%s\n", colorGreen, colorReset, colorGreen, message, colorReset)
}

func printError(message string) {
	fmt.Printf("%s✗%s %s%s%s\n", colorRed, colorReset, colorRed, message, colorReset)
}

func printProgress(message string) {
	fmt.Printf("%s>%s %s%s%s\n", colorYellow, colorReset, colorYellow, message, colorReset)
}
