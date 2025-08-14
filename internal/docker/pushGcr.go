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
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
)

type FancyLogger struct {
	startTime     time.Time
	lastUpdate    map[string]float64
	layerWidth    int
	spinnerStates []string
	spinnerIndex  int
}

func NewFancyLogger() *FancyLogger {
	return &FancyLogger{
		startTime:     time.Now(),
		lastUpdate:    make(map[string]float64),
		layerWidth:    12, // Truncate long layer IDs
		spinnerStates: []string{"⣷", "⣯", "⣟", "⡿", "⢿", "⣻", "⣽", "⣾"},
	}
}

func (l *FancyLogger) getSpinner() string {
	l.spinnerIndex = (l.spinnerIndex + 1) % len(l.spinnerStates)
	return l.spinnerStates[l.spinnerIndex]
}

func (l *FancyLogger) logHeader(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n", colorBlue, time.Since(l.startTime).Round(time.Millisecond), "⚡", colorCyan, message, colorReset)
}

func (l *FancyLogger) logSuccess(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n", colorGreen, time.Since(l.startTime).Round(time.Millisecond), "✓", colorGreen, message, colorReset)
}

func (l *FancyLogger) logProgress(layer, operation string, percent float64) {
	// Truncate layer ID for display
	displayLayer := layer
	if len(layer) > l.layerWidth {
		displayLayer = layer[:l.layerWidth] + "..."
	}

	// Only update if:
	// - Operation changed
	// - Progress increased by at least 5%
	// - It's the first update for this layer
	lastPercent, exists := l.lastUpdate[layer]
	if !exists || operation != "" || percent-lastPercent >= 5 || percent == 100 {
		bar := l.progressBar(percent)
		spinner := l.getSpinner()
		fmt.Printf("\r%s[%s] %s %s: %s %s %s",
			colorYellow,
			time.Since(l.startTime).Round(time.Millisecond),
			spinner,
			colorMagenta+displayLayer+colorReset,
			operation,
			bar,
			fmt.Sprintf("%5.1f%%", percent))

		if percent == 100 {
			fmt.Println() // New line when complete
		}
		l.lastUpdate[layer] = percent
	}
}

func (l *FancyLogger) progressBar(percent float64) string {
	const width = 20
	completed := int(percent / 5)
	if completed > width {
		completed = width
	}
	return fmt.Sprintf("%s%s%s%s",
		colorGreen,
		strings.Repeat("█", completed),
		colorWhite,
		strings.Repeat("░", width-completed))
}

func (l *FancyLogger) logFinal(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n", colorMagenta, time.Since(l.startTime).Round(time.Millisecond), "✨", colorMagenta, message, colorReset)
}

// Helper function to parse image name and tag
func parseImageName(imageNameWithTag string) (string, string) {
	parts := strings.Split(imageNameWithTag, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return imageNameWithTag, "latest"
}

// Helper function to build tagged image name
func buildTaggedImageName(projectID, imageName, imageTag string) string {
	// If image already contains registry info, use as-is
	if strings.Contains(imageName, "gcr.io") || strings.Contains(imageName, "docker.pkg.dev") {
		return fmt.Sprintf("%s:%s", imageName, imageTag)
	}
	// Otherwise prepend gcr.io registry
	return fmt.Sprintf("gcr.io/%s/%s:%s", projectID, imageName, imageTag)
}

// Helper function to build GCR console link
func buildGCRLink(projectID, imageName string) string {
	registryType := "gcr"
	if strings.Contains(imageName, "docker.pkg.dev") {
		registryType = "artifacts/repository"
	}
	return fmt.Sprintf("https://console.cloud.google.com/%s/images/%s/%s?project=%s",
		registryType, projectID, imageName, projectID)
}

func PushImageToGCR(projectID, imageNameWithTag string) error {
	logger := NewFancyLogger()
	ctx := context.Background()

	logger.logHeader("Starting GCR image push")
	defer func() {
		logger.logHeader("Push operation completed")
	}()

	// Authentication
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return fmt.Errorf("%sauthentication failed%s: %v", colorRed, colorReset, err)
	}
	logger.logSuccess("Google Cloud authenticated")

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
	imageName, imageTag := parseImageName(imageNameWithTag)
	taggedImage := buildTaggedImageName(projectID, imageName, imageTag)

	if err := dockerClient.ImageTag(ctx, fmt.Sprintf("%s:%s", imageName, imageTag), taggedImage); err != nil {
		return fmt.Errorf("%simage tagging failed%s: %v", colorRed, colorReset, err)
	}
	logger.logSuccess(fmt.Sprintf("Image tagged: %s%s%s", colorCyan, taggedImage, colorReset))

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

	// Fancy progress logging
	logger.logHeader("Pushing image layers")
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

		if event.ID != "" {
			percent := 0.0
			if event.Progress != nil && event.Progress.Total > 0 {
				percent = float64(event.Progress.Current) / float64(event.Progress.Total) * 100
			}
			logger.logProgress(event.ID, event.Status, percent)
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

	logger.logFinal(fmt.Sprintf("Image pushed successfully!"))
	logger.logFinal(fmt.Sprintf("View in console: %s%s%s", colorCyan, link, colorReset))
	logger.logFinal(fmt.Sprintf("Image reference: %s%s%s", colorCyan, taggedImage, colorReset))

	return nil
}
