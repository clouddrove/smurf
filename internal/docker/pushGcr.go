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

// Color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

type ColorfulLogger struct {
	startTime time.Time
}

func NewColorfulLogger() *ColorfulLogger {
	return &ColorfulLogger{startTime: time.Now()}
}

func (l *ColorfulLogger) logStep(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n",
		colorBlue,
		time.Since(l.startTime).Round(time.Millisecond),
		"→",
		colorCyan,
		message,
		colorReset)
}

func (l *ColorfulLogger) logSuccess(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n",
		colorGreen,
		time.Since(l.startTime).Round(time.Millisecond),
		"✓",
		colorGreen,
		message,
		colorReset)
}

func (l *ColorfulLogger) logLayerPushed(layerID string) {
	fmt.Printf("%s[%s] %s %s%s pushed%s\n",
		colorYellow,
		time.Since(l.startTime).Round(time.Millisecond),
		"⬆",
		colorCyan,
		layerID[:12]+"...",
		colorReset)
}

func PushImageToGCR(projectID, imageNameWithTag string) error {
	logger := NewColorfulLogger()
	ctx := context.Background()

	logger.logStep("Starting image push to GCR")

	// Authentication
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("%sauthentication failed%s: %v", colorRed, colorReset, err)
	}
	logger.logSuccess("Authenticated with Google Cloud")

	token, err := creds.TokenSource.Token()
	if err != nil {
		return fmt.Errorf("%stoken acquisition failed%s: %v", colorRed, colorReset, err)
	}

	// Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("%sdocker client creation failed%s: %v", colorRed, colorReset, err)
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
		return fmt.Errorf("%simage tagging failed%s: %v", colorRed, colorReset, err)
	}
	logger.logSuccess(fmt.Sprintf("Tagged image: %s%s%s", colorCyan, taggedImage, colorReset))

	// Image push
	authConfig := registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: "https://gcr.io",
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return fmt.Errorf("%sauth encoding failed%s: %v", colorRed, colorReset, err)
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, image.PushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return fmt.Errorf("%spush failed%s: %v", colorRed, colorReset, err)
	}
	defer pushResponse.Close()

	logger.logStep("Starting image push")
	dec := json.NewDecoder(pushResponse)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("%spush response failed%s: %v", colorRed, colorReset, err)
		}
		if event.Error != nil {
			return fmt.Errorf("%spush failed%s: %v", colorRed, colorReset, event.Error)
		}

		// Only log when a layer is fully pushed
		if event.Status == "Pushed" && event.ID != "" {
			logger.logLayerPushed(event.ID)
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

	logger.logSuccess("Image pushed successfully")
	logger.logSuccess(fmt.Sprintf("View in console: %s%s%s", colorCyan, link, colorReset))
	logger.logSuccess(fmt.Sprintf("Image reference: %s%s%s", colorCyan, taggedImage, colorReset))

	return nil
}
