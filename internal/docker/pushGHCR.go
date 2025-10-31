package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// In docker package
type LoginOptions struct {
	Registry string
	Username string
	Password string
}

func Login(opts LoginOptions) error {
	// Implementation for docker login
	cmd := exec.Command("docker", "login", opts.Registry, "-u", opts.Username, "-p", opts.Password)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func PushToGHCR(opts PushOptions) error {
	fmt.Printf("Initializing Docker client for GHCR...\n")

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("❌ Failed to initialize Docker client\n")
		return fmt.Errorf("failed to initialize Docker client: %v", err)
	}
	defer cli.Close()

	fmt.Printf("Preparing GHCR authentication...\n")

	authConfig := registry.AuthConfig{
		Username:      os.Getenv("GITHUB_USERNAME"),
		Password:      os.Getenv("GITHUB_TOKEN"),
		ServerAddress: "ghcr.io",
	}

	if authConfig.Username == "" || authConfig.Password == "" {
		fmt.Printf("❌ GitHub credentials not found\n")
		return fmt.Errorf("GITHUB_USERNAME and GITHUB_TOKEN environment variables are required for GHCR")
	}

	if !strings.HasPrefix(opts.ImageName, "ghcr.io/") {
		fmt.Printf("❌ Image name must be for GHCR registry\n")
		return fmt.Errorf("image name must start with 'ghcr.io/' for GitHub Container Registry")
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		fmt.Printf("❌ GHCR authentication preparation failed: %v\n", err)
		return fmt.Errorf("GHCR authentication preparation failed: %v", err)
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	fmt.Printf("Pushing image to GitHub Container Registry: %s\n", opts.ImageName)
	fmt.Printf("─────────────────────────────────────────────────────────────\n")

	pushResp, err := cli.ImagePush(ctx, opts.ImageName, image.PushOptions{
		RegistryAuth: authStr,
	})

	if err != nil {
		fmt.Printf("❌ Failed to push image %s to GHCR, error: %v\n", opts.ImageName, err)
		return fmt.Errorf("failed to push image to GHCR: %v", err)
	}
	defer pushResp.Close()

	decoder := json.NewDecoder(pushResp)
	currentLayer := ""
	layerStatus := make(map[string]string)

	for {
		var msg jsonMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		if msg.Error != "" {
			fmt.Printf("❌ GHCR push error: %s\n", msg.Error)
			return fmt.Errorf("failed to push image to GHCR: %v", msg.Error)
		}

		// Handle layer changes
		if msg.ID != "" && msg.ID != currentLayer {
			if currentLayer != "" {
				// Show completion status for previous layer
				status := layerStatus[currentLayer]
				if status == "" {
					status = "completed"
				}
				fmt.Printf("   ✅ %s\n", status)
			}
			currentLayer = msg.ID
			fmt.Printf("Layer: %s\n", msg.ID)
		}

		// Update and show meaningful status changes
		if msg.Status != "" && currentLayer != "" {
			// Only show important status updates
			showStatus := false
			var displayText string

			switch {
			case strings.Contains(msg.Status, "Pushing"):
				if !strings.Contains(msg.Progress, "MB") {
					layerStatus[currentLayer] = "uploading"
					displayText = "📤 Uploading"
					showStatus = true
				}
			case strings.Contains(msg.Status, "Pushed"):
				layerStatus[currentLayer] = "pushed"
				displayText = "✅ Pushed"
				showStatus = true
			case strings.Contains(msg.Status, "Layer already exists"):
				layerStatus[currentLayer] = "cached"
				displayText = "⚡ Already exists"
				showStatus = true
			case strings.Contains(msg.Status, "Mounted"):
				layerStatus[currentLayer] = "mounted"
				displayText = "🔗 Mounted from cache"
				showStatus = true
			case strings.Contains(msg.Status, "Verifying Checksum"):
				layerStatus[currentLayer] = "verifying"
				displayText = "🔍 Verifying checksum"
				showStatus = true
			}

			if showStatus {
				fmt.Printf("   %s\n", displayText)
			}
		}

		// Show final digest
		if msg.ID == "" && strings.Contains(msg.Status, "Digest:") {
			if currentLayer != "" {
				fmt.Printf("   ✅ completed\n")
				currentLayer = ""
			}
			fmt.Printf("📦 %s\n", msg.Status)
		}
	}

	// Complete the last layer if needed
	if currentLayer != "" {
		status := layerStatus[currentLayer]
		if status == "" {
			status = "completed"
		}
		fmt.Printf("   ✅ %s\n", status)
	}

	fmt.Printf("─────────────────────────────────────────────────────────────\n")
	fmt.Printf("✅ Successfully pushed to GitHub Container Registry\n")
	fmt.Printf("📦 Image: %s\n", opts.ImageName)

	parts := strings.Split(opts.ImageName, "/")
	if len(parts) >= 3 {
		repoParts := strings.Split(parts[2], ":")
		repoName := repoParts[0]
		fmt.Printf("🌐 View at: https://github.com/%s/pkgs/container/%s\n", parts[1], repoName)
	}

	return nil
}
