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

// LogLevel type for different log levels
type LogLevel int

const (
	INFO LogLevel = iota
	SUCCESS
	WARNING
	ERROR
	DEBUG
)

// Color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
)

// Logger structure
type Logger struct {
	startTime time.Time
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{startTime: time.Now()}
}

// log prints formatted log messages
func (l *Logger) log(level LogLevel, message string, args ...interface{}) {
	timestamp := time.Since(l.startTime).Round(time.Millisecond)
	msg := fmt.Sprintf(message, args...)

	var prefix, color string
	switch level {
	case SUCCESS:
		prefix = "✓"
		color = colorGreen
	case WARNING:
		prefix = "⚠"
		color = colorYellow
	case ERROR:
		prefix = "✗"
		color = colorRed
	case DEBUG:
		prefix = "•"
		color = colorMagenta
	default: // INFO
		prefix = "»"
		color = colorBlue
	}

	fmt.Printf("%s[%s] %s %s%s%s\n", color, timestamp, prefix, color, msg, colorReset)
}

// PushImageToGCR pushes the specified Docker image to Google Container Registry
func PushImageToGCR(projectID, imageNameWithTag string) error {
	logger := NewLogger()
	ctx := context.Background()

	// Parse image name and tag
	imageName := imageNameWithTag
	imageTag := "latest"
	if parts := strings.Split(imageNameWithTag, ":"); len(parts) == 2 {
		imageName = parts[0]
		imageTag = parts[1]
	}

	logger.log(INFO, "Authenticating with Google Cloud...")
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		logger.log(ERROR, "Failed to authenticate with Google Cloud")
		return fmt.Errorf("authentication failed: %v", err)
	}
	logger.log(SUCCESS, "Authenticated with Google Cloud")

	logger.log(INFO, "Obtaining access token...")
	token, err := creds.TokenSource.Token()
	if err != nil {
		logger.log(ERROR, "Failed to obtain access token")
		return fmt.Errorf("token acquisition failed: %v", err)
	}
	logger.log(SUCCESS, "Access token obtained")

	logger.log(INFO, "Creating Docker client...")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.log(ERROR, "Failed to create Docker client")
		return fmt.Errorf("docker client creation failed: %v", err)
	}
	logger.log(SUCCESS, "Docker client created")

	logger.log(INFO, "Tagging the image...")
	registryHost := ""
	if !strings.Contains(imageName, "gcr.io") && !strings.Contains(imageName, "docker.pkg.dev") {
		registryHost = fmt.Sprintf("gcr.io/%s/", projectID)
	}

	sourceImage := fmt.Sprintf("%s:%s", imageName, imageTag)
	taggedImage := sourceImage
	if registryHost != "" {
		taggedImage = fmt.Sprintf("%s%s:%s", registryHost, imageName, imageTag)
	}

	if err := dockerClient.ImageTag(ctx, sourceImage, taggedImage); err != nil {
		logger.log(ERROR, "Failed to tag the image")
		return fmt.Errorf("image tagging failed: %v", err)
	}
	logger.log(SUCCESS, "Image tagged as: %s", taggedImage)

	logger.log(INFO, "Pushing the image to GCR...")
	authConfig := registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: "https://gcr.io",
	}
	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		logger.log(ERROR, "Failed to encode authentication credentials")
		return fmt.Errorf("auth encoding failed: %v", err)
	}

	pushResponse, err := dockerClient.ImagePush(ctx, taggedImage, image.PushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		logger.log(ERROR, "Failed to push the image")
		return fmt.Errorf("image push failed: %v", err)
	}
	defer pushResponse.Close()

	// Enhanced structured push logging
	var (
		lastLayer     string
		lastOperation string
	)
	dec := json.NewDecoder(pushResponse)
	for {
		var event jsonmessage.JSONMessage
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			logger.log(ERROR, "Failed to read push response")
			return fmt.Errorf("push response read failed: %v", err)
		}
		if event.Error != nil {
			logger.log(ERROR, "Push failed: %v", event.Error)
			return fmt.Errorf("push failed: %v", event.Error)
		}

		if event.ID != "" || event.Status != "" {
			// Log layer changes
			if event.ID != "" && event.ID != lastLayer {
				logger.log(INFO, "Processing layer: %s", event.ID)
				lastLayer = event.ID
			}

			// Log operation changes
			if event.Status != "" && event.Status != lastOperation {
				progress := ""
				if event.Progress != nil {
					progress = fmt.Sprintf(" (%d/%d)", event.Progress.Current, event.Progress.Total)
				}
				logger.log(DEBUG, "%s%s%s", event.Status, progress)
				lastOperation = event.Status
			}
		}
	}

	// Construct and log success message
	registryType := "gcr"
	if strings.Contains(imageName, "docker.pkg.dev") {
		registryType = "artifacts/repository"
	}
	link := fmt.Sprintf("https://console.cloud.google.com/%s/images/%s/%s?project=%s",
		registryType, projectID, imageName, projectID)

	// GitHub Actions output
	if outputPath := os.Getenv("GITHUB_OUTPUT"); outputPath != "" {
		file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err == nil {
			fmt.Fprintf(file, "image_url=%s\n", link)
			file.Close()
		}
	}

	logger.log(SUCCESS, "Image successfully pushed to GCR")
	logger.log(INFO, "Image URL: %s", link)
	logger.log(INFO, "Full image reference: %s", taggedImage)

	return nil
}
