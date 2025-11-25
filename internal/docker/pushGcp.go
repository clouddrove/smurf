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

// Constants
const (
	// Google Cloud Platform authentication scope
	GoogleCloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
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

func (l *ColorfulLogger) log(level, symbol, color, message string) {
	fmt.Printf("%s[%s] %s %s%s%s\n",
		color,
		time.Since(l.startTime).Round(time.Millisecond),
		symbol,
		color,
		message,
		colorReset)
}

func (l *ColorfulLogger) logStep(message string) {
	l.log("step", "→", colorBlue, message)
}

func (l *ColorfulLogger) logSuccess(message string) {
	l.log("success", "✓", colorGreen, message)
}

func (l *ColorfulLogger) logWarning(message string) {
	l.log("warning", "⚠", colorYellow, message)
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

// AuthProvider handles Google Cloud authentication
type AuthProvider struct {
	logger *ColorfulLogger
}

func NewAuthProvider() *AuthProvider {
	return &AuthProvider{
		logger: NewColorfulLogger(),
	}
}

// VerifyGCloudAuth verifies Google Cloud authentication with multiple fallbacks
func (a *AuthProvider) VerifyGCloudAuth() error {
	ctx := context.Background()
	a.logger.logStep("Checking Google Cloud authentication...")

	authMethods := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"gcloud CLI", a.verifyGCloudCLI},
		{"service account credentials", a.verifyServiceAccount},
		{"default credentials", a.verifyDefaultCredentials},
	}

	for _, method := range authMethods {
		if err := method.fn(ctx); err == nil {
			a.logger.logSuccess(fmt.Sprintf("Authentication verified via %s", method.name))
			return nil
		}
	}

	a.logger.logWarning("No valid Google Cloud authentication found")
	return fmt.Errorf(`google Cloud authentication required. Please run:

  gcloud auth login

Or set service account credentials:

  export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"`)
}

func (a *AuthProvider) verifyGCloudCLI(ctx context.Context) error {
	token, err := a.getGcloudAccessToken()
	return a.validateToken(token, err)
}

func (a *AuthProvider) verifyServiceAccount(ctx context.Context) error {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		return fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		return err
	}

	creds, err := google.CredentialsFromJSON(ctx, data, GoogleCloudPlatformScope)
	if err != nil {
		return err
	}

	return a.validateCredentials(creds)
}

func (a *AuthProvider) verifyDefaultCredentials(ctx context.Context) error {
	creds, err := google.FindDefaultCredentials(ctx, GoogleCloudPlatformScope)
	if err != nil {
		return err
	}
	return a.validateCredentials(creds)
}

func (a *AuthProvider) validateToken(token string, err error) error {
	if err != nil || token == "" {
		return fmt.Errorf("invalid token")
	}
	return nil
}

func (a *AuthProvider) validateCredentials(creds *google.Credentials) error {
	if creds == nil {
		return fmt.Errorf("credentials are nil")
	}
	token, err := creds.TokenSource.Token()
	if err != nil {
		return err
	}
	return a.validateToken(token.AccessToken, nil)
}

// getGcloudAccessToken tries to get access token using gcloud CLI
func (a *AuthProvider) getGcloudAccessToken() (string, error) {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getAuthConfig tries multiple authentication methods in order
func (a *AuthProvider) getAuthConfig(serverAddress string) (registry.AuthConfig, error) {
	authMethods := []func(string) (registry.AuthConfig, error){
		a.getDockerConfigAuth,
		a.getGCloudTokenAuth,
		a.getServiceAccountAuth,
		a.getDefaultCredentialsAuth,
	}

	for _, method := range authMethods {
		if auth, err := method(serverAddress); err == nil {
			return auth, nil
		}
	}

	return registry.AuthConfig{}, fmt.Errorf("no valid authentication methods found")
}

func (a *AuthProvider) getGCloudTokenAuth(serverAddress string) (registry.AuthConfig, error) {
	token, err := a.getGcloudAccessToken()
	if err != nil {
		return registry.AuthConfig{}, err
	}
	return registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token,
		ServerAddress: serverAddress,
	}, nil
}

func (a *AuthProvider) getServiceAccountAuth(serverAddress string) (registry.AuthConfig, error) {
	ctx := context.Background()
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		return registry.AuthConfig{}, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	data, err := os.ReadFile(credsPath)
	if err != nil {
		return registry.AuthConfig{}, err
	}

	return a.getAuthFromCredentials(ctx, data, serverAddress)
}

func (a *AuthProvider) getDefaultCredentialsAuth(serverAddress string) (registry.AuthConfig, error) {
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, GoogleCloudPlatformScope)
	if err != nil {
		return registry.AuthConfig{}, err
	}
	return a.getAuthFromCredentials(ctx, creds.JSON, serverAddress)
}

func (a *AuthProvider) getAuthFromCredentials(ctx context.Context, jsonData []byte, serverAddress string) (registry.AuthConfig, error) {
	creds, err := google.CredentialsFromJSON(ctx, jsonData, GoogleCloudPlatformScope)
	if err != nil {
		return registry.AuthConfig{}, err
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return registry.AuthConfig{}, err
	}

	return registry.AuthConfig{
		Username:      "oauth2accesstoken",
		Password:      token.AccessToken,
		ServerAddress: serverAddress,
	}, nil
}

// getDockerConfigAuth tries to get auth from Docker config (set by gcloud auth configure-docker)
func (a *AuthProvider) getDockerConfigAuth(serverAddress string) (registry.AuthConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return registry.AuthConfig{}, err
	}

	dockerConfigPath := filepath.Join(homeDir, ".docker", "config.json")
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

	authData, exists := config.Auths[serverAddress]
	if !exists || authData.Auth == "" {
		return registry.AuthConfig{}, fmt.Errorf("no auth found for %s", serverAddress)
	}

	decoded, err := base64.StdEncoding.DecodeString(authData.Auth)
	if err != nil {
		return registry.AuthConfig{}, err
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return registry.AuthConfig{}, fmt.Errorf("invalid auth format")
	}

	return registry.AuthConfig{
		Username:      parts[0],
		Password:      parts[1],
		ServerAddress: serverAddress,
	}, nil
}

// Helper functions
func extractServerAddress(imageName string) string {
	parts := strings.Split(imageName, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return imageName
}

// PushImageToGCR pushes image to Google Container Registry/Artifact Registry
func PushImageToGCR(projectID, imageNameWithTag string) error {
	logger := NewColorfulLogger()
	authProvider := NewAuthProvider()
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
	authConfig, err := authProvider.getAuthConfig(serverAddress)
	if err != nil {
		return fmt.Errorf("%sauthentication failed%s: %v", colorRed, colorReset, err)
	}

	encodedAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return fmt.Errorf("%sauth encoding failed%s: %v", colorRed, colorReset, err)
	}

	return pushImageGCP(ctx, dockerClient, targetImage, encodedAuth, logger)
}

func pushImageGCP(ctx context.Context, dockerClient *client.Client, imageName, encodedAuth string, logger *ColorfulLogger) error {
	pushResponse, err := dockerClient.ImagePush(ctx, imageName, image.PushOptions{
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
	logger.logSuccess(fmt.Sprintf("Image reference: %s%s%s", colorCyan, imageName, colorReset))

	return nil
}

// VerifyGCloudAuth maintains backward compatibility
func VerifyGCloudAuth() error {
	return NewAuthProvider().VerifyGCloudAuth()
}
