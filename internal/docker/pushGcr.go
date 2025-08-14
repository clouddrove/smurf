package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"golang.org/x/oauth2/google"
)

type MinimalLogger struct {
	startTime time.Time
}

func NewMinimalLogger() *MinimalLogger {
	return &MinimalLogger{startTime: time.Now()}
}

func (l *MinimalLogger) log(message string) {
	fmt.Printf("[%s] %s\n", time.Since(l.startTime).Round(time.Millisecond), message)
}

func PushImageToGCR(projectID, imageNameWithTag string) error {
	logger := NewMinimalLogger()
	ctx := context.Background()

	logger.log("Starting image push to GCR")

	// Authentication
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}
	logger.log("Authenticated with Google Cloud")

	token, err := creds.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("token acquisition failed: %v", err)
	}

	// Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker client creation failed: %v", err)
	}

	// Image tagging
	parts := strings.Split(imageNameWithTag, ":")
	imageName, imageTag := parts[0], "latest"
	if len(parts) == 2 {
		imageTag = parts[1]
	}

	taggedImage := imageNameWithTag
	if !strings.Contains(imageName, "gcr.io") && !strings.Contains(imageName, "docker.pkg.dev") {
		taggedImage = fmt.Sprintf("gcr.io/%s/%s:%s", projectID, imageName, imageTag)
	}

	if err := dockerClient.ImageTag(ctx, imageNameWithTag, taggedImage); err != nil {
		return fmt.Errorf("image tagging failed: %v", err)
	}
	logger.log(fmt.Sprintf("Tagged image: %s", taggedImage))

	// Image push
	authConfig := registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: "https://gcr.io",
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return fmt.Errorf("auth encoding failed: %v", err)
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, image.PushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return fmt.Errorf("push failed: %v", err)
	}
	defer pushResponse.Close()

	logger.log("Starting image push")
	dec := json.NewDecoder(pushResponse)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("push response failed: %v", err)
		}
		if event.Error != nil {
			return fmt.Errorf("push failed: %v", event.Error)
		}

		// Only log when a layer is fully pushed
		if event.Status == "Pushed" {
			logger.log(fmt.Sprintf("Layer %s pushed", event.ID[:12]+"..."))
		}
	}

	// Final output
	registryType := "gcr"
	if strings.Contains(imageName, "docker.pkg.dev") {
		registryType = "artifacts/repository"
	}
	link := fmt.Sprintf("https://console.cloud.google.com/%s/images/%s/%s?project=%s",
		registryType, projectID, imageName, projectID)

	if outputPath := os.Getenv("GITHUB_OUTPUT"); outputPath != "" {
		file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintf(file, "image_url=%s\n", link)
			file.Close()
		}
	}

	logger.log("Image pushed successfully")
	logger.log(fmt.Sprintf("View in console: %s", link))
	logger.log(fmt.Sprintf("Image reference: %s", taggedImage))

	return nil
}
