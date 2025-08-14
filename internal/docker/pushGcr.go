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
	startTime    time.Time
	lastLayer    string
	lastProgress string
}

func NewMinimalLogger() *MinimalLogger {
	return &MinimalLogger{startTime: time.Now()}
}

func (l *MinimalLogger) logStep(message string) {
	fmt.Printf("[%s] → %s\n", time.Since(l.startTime).Round(time.Millisecond), message)
}

func (l *MinimalLogger) logSuccess(message string) {
	fmt.Printf("[%s] ✓ %s\n", time.Since(l.startTime).Round(time.Millisecond), message)
}

func (l *MinimalLogger) logProgress(layer, operation string, current, total int64) {
	progress := ""
	if total > 0 {
		progress = fmt.Sprintf(" (%.1f%%)", float64(current)/float64(total)*100)
	}
	msg := fmt.Sprintf("%s: %s%s", layer, operation, progress)

	if msg != l.lastProgress {
		fmt.Printf("[%s] ↳ %s\n", time.Since(l.startTime).Round(time.Millisecond), msg)
		l.lastProgress = msg
	}
}

func (l *MinimalLogger) logFinal(message string) {
	fmt.Printf("[%s] ★ %s\n", time.Since(l.startTime).Round(time.Millisecond), message)
}

func parseImageName(imageNameWithTag string) (string, string) {
	parts := strings.Split(imageNameWithTag, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return imageNameWithTag, "latest"
}

func buildTaggedImageName(projectID, imageName, imageTag string) string {
	if strings.Contains(imageName, "gcr.io") || strings.Contains(imageName, "docker.pkg.dev") {
		return fmt.Sprintf("%s:%s", imageName, imageTag)
	}
	return fmt.Sprintf("gcr.io/%s/%s:%s", projectID, imageName, imageTag)
}

func buildGCRLink(projectID, imageName string) string {
	registryType := "gcr"
	if strings.Contains(imageName, "docker.pkg.dev") {
		registryType = "artifacts/repository"
	}
	return fmt.Sprintf("https://console.cloud.google.com/%s/images/%s/%s?project=%s",
		registryType, projectID, imageName, projectID)
}

func PushImageToGCR(projectID, imageNameWithTag string) error {
	logger := NewMinimalLogger()
	ctx := context.Background()

	// Initial setup
	logger.logStep("Starting image push to GCR")
	defer func() {
		logger.logStep("Push operation completed")
	}()

	// Authentication
	logger.logStep("Authenticating with Google Cloud")
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}
	logger.logSuccess("Google Cloud authenticated")

	logger.logStep("Obtaining access token")
	token, err := creds.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("token acquisition failed: %v", err)
	}

	// Docker client
	logger.logStep("Creating Docker client")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker client creation failed: %v", err)
	}

	// Image tagging
	imageName, imageTag := parseImageName(imageNameWithTag)
	taggedImage := buildTaggedImageName(projectID, imageName, imageTag)

	logger.logStep(fmt.Sprintf("Tagging image as: %s", taggedImage))
	if err := dockerClient.ImageTag(ctx, fmt.Sprintf("%s:%s", imageName, imageTag), taggedImage); err != nil {
		return fmt.Errorf("image tagging failed: %v", err)
	}
	logger.logSuccess("Image successfully tagged")

	// Image push
	logger.logStep("Preparing to push image")
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
		return fmt.Errorf("image push failed: %v", err)
	}
	defer pushResponse.Close()

	// Minimal push progress logging
	logger.logStep("Starting image push")
	dec := json.NewDecoder(pushResponse)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("push response read failed: %v", err)
		}
		if event.Error != nil {
			return fmt.Errorf("push failed: %v", event.Error)
		}

		if event.ID != "" && event.Status != "" {
			current := int64(0)
			total := int64(0)
			if event.Progress != nil {
				current = event.Progress.Current
				total = event.Progress.Total
			}

			// Only log significant events
			if event.ID != logger.lastLayer ||
				(total > 0 && (current == 0 || current == total || current%(total/10) == 0)) {
				logger.logProgress(event.ID, event.Status, current, total)
				logger.lastLayer = event.ID
			}
		}
	}

	// Final output
	link := buildGCRLink(projectID, imageName)
	if outputPath := os.Getenv("GITHUB_OUTPUT"); outputPath != "" {
		if file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			fmt.Fprintf(file, "image_url=%s\n", link)
			file.Close()
		}
	}

	logger.logFinal(fmt.Sprintf("Image successfully pushed to GCR"))
	logger.logFinal(fmt.Sprintf("View in console: %s", link))
	logger.logFinal(fmt.Sprintf("Image reference: %s", taggedImage))

	return nil
}
