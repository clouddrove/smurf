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

	// Common gcloud binary paths
	gcloudBinaryUnix     = "/usr/bin/gcloud"
	gcloudBinaryHomebrew = "/usr/local/bin/gcloud"
	gcloudBinarySDK      = "/opt/google-cloud-sdk/bin/gcloud"

	// Safe PATH directories for Unix-like systems
	safeUnixPaths = "/usr/local/bin:/usr/bin:/bin"

	// Safe PATH directories for Windows
	safeWindowsPaths = `C:\Windows\System32;C:\Windows;`
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

// getGcloudAccessToken tries to get access token using gcloud CLI with security fixes
func (a *AuthProvider) getGcloudAccessToken() (string, error) {
	// Find gcloud binary securely
	gcloudPath, err := a.findGcloudBinary()
	if err != nil {
		return "", fmt.Errorf("gcloud binary not found: %v", err)
	}

	// Use absolute path and explicit arguments
	cmd := exec.Command(gcloudPath, "auth", "print-access-token")

	// Set secure environment to prevent injection
	cmd.Env = a.getSecureEnvironment()

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gcloud auth failed: %v", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// findGcloudBinary finds gcloud binary securely
func (a *AuthProvider) findGcloudBinary() (string, error) {
	// Check common absolute paths first
	possiblePaths := []string{
		gcloudBinaryUnix,
		gcloudBinaryHomebrew,
		gcloudBinarySDK,
	}

	// Add user home directory path if available
	if homeDir, err := os.UserHomeDir(); err == nil {
		possiblePaths = append(possiblePaths,
			filepath.Join(homeDir, "google-cloud-sdk", "bin", "gcloud"),
		)
	}

	for _, path := range possiblePaths {
		if a.isSafeBinary(path) {
			return path, nil
		}
	}

	// Fallback to which/where command with secure execution
	return a.findGcloudWithWhich()
}

// findGcloudWithWhich uses which/where command securely
func (a *AuthProvider) findGcloudWithWhich() (string, error) {
	var cmd *exec.Cmd

	// Use OS-specific commands with absolute paths
	if a.isWindows() {
		cmd = exec.Command("where.exe", "gcloud")
	} else {
		// Use absolute paths for which command
		whichPaths := []string{"/usr/bin/which", "/bin/which"}
		for _, whichPath := range whichPaths {
			if _, err := os.Stat(whichPath); err == nil {
				cmd = exec.Command(whichPath, "gcloud")
				break
			}
		}
		if cmd == nil {
			return "", fmt.Errorf("which command not found")
		}
	}

	// Set secure environment
	cmd.Env = a.getSecureEnvironment()

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gcloud not found in PATH: %v", err)
	}

	path := strings.TrimSpace(string(output))
	if path == "" {
		return "", fmt.Errorf("gcloud binary not found")
	}

	// Validate the found path is safe
	if !a.isSafeBinary(path) {
		return "", fmt.Errorf("gcloud binary path is not safe: %s", path)
	}

	return path, nil
}

// isSafeBinary checks if a binary path is safe to execute
func (a *AuthProvider) isSafeBinary(path string) bool {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check if path is absolute
	if !filepath.IsAbs(cleanPath) {
		return false
	}

	// Check if file exists and is executable
	info, err := os.Stat(cleanPath)
	if err != nil {
		return false
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return false
	}

	// Check for dangerous patterns in path
	dangerousPatterns := []string{
		"/tmp/",
		"/var/tmp/",
		"/dev/",
		"/proc/",
		"..",
		"./",
		"~",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(cleanPath, pattern) {
			return false
		}
	}

	// Check file permissions (should not be world-writable)
	if info.Mode().Perm()&0002 != 0 {
		return false
	}

	return true
}

// getSecureEnvironment returns a clean environment with only safe variables
func (a *AuthProvider) getSecureEnvironment() []string {
	// Start with minimal safe environment
	env := []string{}

	// Only include essential, safe environment variables
	safeVars := []string{
		"HOME",
		"USER",
		"TMPDIR",
		"TEMP",
		"TMP",
		"USERPROFILE", // Windows
	}

	for _, key := range safeVars {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}

	// Add safe PATH
	if a.isWindows() {
		env = append(env, "PATH="+safeWindowsPaths)
	} else {
		env = append(env, "PATH="+safeUnixPaths)
	}

	return env
}

// isWindows checks if running on Windows
func (a *AuthProvider) isWindows() bool {
	return os.PathSeparator == '\\' && strings.Contains(strings.ToLower(os.Getenv("OS")), "windows")
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
