package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

func (l *ColorfulLogger) logWarning(message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n",
		colorYellow,
		time.Since(l.startTime).Round(time.Millisecond),
		"⚠",
		colorYellow,
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

// VerifyGCloudAuth verifies Google Cloud authentication with multiple fallbacks
func VerifyGCloudAuth() error {
	ctx := context.Background()
	logger := NewColorfulLogger()

	logger.logStep("Checking Google Cloud authentication...")

	// Method 1: Check if we can use gcloud CLI
	if token, err := getGcloudAccessToken(); err == nil && token != "" {
		logger.logSuccess("Authentication verified via gcloud CLI")
		return nil
	}

	// Method 2: Check service account credentials
	if credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsPath != "" {
		if _, err := os.Stat(credsPath); err == nil {
			data, err := os.ReadFile(credsPath)
			if err == nil {
				_, err := google.CredentialsFromJSON(ctx, data, "https://www.googleapis.com/auth/cloud-platform")
				if err == nil {
					logger.logSuccess("Authentication verified via service account credentials")
					return nil
				}
			}
		}
	}

	// Method 3: Check default credentials
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err == nil && creds != nil {
		// Try to get a token to verify
		token, err := creds.TokenSource.Token()
		if err == nil && token != nil && token.Valid() {
			logger.logSuccess("Authentication verified via default credentials")
			return nil
		}
	}

	logger.logWarning("No valid Google Cloud authentication found")
	return fmt.Errorf(`Google Cloud authentication required. Please run:

  gcloud auth login

Or set service account credentials:

  export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"`)
}

// getGcloudAccessToken tries to get access token using gcloud CLI
func getGcloudAccessToken() (string, error) {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getDockerConfigAuth tries to get auth from Docker config (set by gcloud auth configure-docker)
func getDockerConfigAuth(serverAddress string) (registry.AuthConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return registry.AuthConfig{}, err
	}

	dockerConfigPath := filepath.Join(homeDir, ".docker", "config.json")
	if _, err := os.Stat(dockerConfigPath); err != nil {
		return registry.AuthConfig{}, err
	}

	data, err := os.ReadFile(dockerConfigPath)
	if err != nil {
		return registry.AuthConfig{}, err
	}

	var config struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return registry.AuthConfig{}, err
	}

	if authData, exists := config.Auths[serverAddress]; exists && authData.Auth != "" {
		decoded, err := base64.StdEncoding.DecodeString(authData.Auth)
		if err != nil {
			return registry.AuthConfig{}, err
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) == 2 {
			return registry.AuthConfig{
				Username:      parts[0],
				Password:      parts[1],
				ServerAddress: serverAddress,
			}, nil
		}
	}

	return registry.AuthConfig{}, fmt.Errorf("no auth found for %s", serverAddress)
}

func PushImageToGCR(projectID, imageNameWithTag string) error {
	logger := NewColorfulLogger()
	ctx := context.Background()

	logger.logStep("Starting image push to Google Container Registry/Artifact Registry")

	// Get Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("%sdocker client creation failed%s: %v", colorRed, colorReset, err)
	}

	// Use the image reference exactly as provided - DO NOT MODIFY IT
	targetImage := imageNameWithTag
	logger.logStep(fmt.Sprintf("Pushing image: %s", targetImage))

	// Determine server address for authentication
	serverAddress := extractServerAddress(targetImage)

	// Get authentication using multiple methods
	authConfig, err := getAuthConfig(serverAddress)
	if err != nil {
		return fmt.Errorf("%sauthentication failed%s: %v", colorRed, colorReset, err)
	}

	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return fmt.Errorf("%sauth encoding failed%s: %v", colorRed, colorReset, err)
	}

	pushResponse, err := dockerClient.ImagePush(ctx, targetImage, image.PushOptions{
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

		if event.Status == "Pushed" && event.ID != "" {
			logger.logLayerPushed(event.ID)
		}
	}

	logger.logSuccess("Image pushed successfully")
	logger.logSuccess(fmt.Sprintf("Image reference: %s%s%s", colorCyan, targetImage, colorReset))

	return nil
}

// getAuthConfig tries multiple authentication methods in order
func getAuthConfig(serverAddress string) (registry.AuthConfig, error) {
	ctx := context.Background()

	// Method 1: Try Docker config (set by gcloud auth configure-docker)
	if auth, err := getDockerConfigAuth(serverAddress); err == nil {
		return auth, nil
	}

	// Method 2: Try gcloud CLI access token
	if token, err := getGcloudAccessToken(); err == nil {
		return registry.AuthConfig{
			Username:      "oauth2accesstoken",
			Password:      token,
			ServerAddress: serverAddress,
		}, nil
	}

	// Method 3: Try service account credentials from environment variable
	if credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsPath != "" {
		if _, err := os.Stat(credsPath); err == nil {
			data, err := os.ReadFile(credsPath)
			if err == nil {
				creds, err := google.CredentialsFromJSON(ctx, data, "https://www.googleapis.com/auth/cloud-platform")
				if err == nil {
					token, err := creds.TokenSource.Token()
					if err == nil {
						return registry.AuthConfig{
							Username:      "oauth2accesstoken",
							Password:      token.AccessToken,
							ServerAddress: serverAddress,
						}, nil
					}
				}
			}
		}
	}

	// Method 4: Try default credentials
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("no valid authentication methods found: %v", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("failed to get authentication token: %v", err)
	}

	return registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: serverAddress,
	}, nil
}

// extractServerAddress extracts server address from image name without modifying it
func extractServerAddress(imageName string) string {
	parts := strings.Split(imageName, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return imageName
}
