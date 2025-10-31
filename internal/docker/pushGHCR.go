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
	layerStatus := make(map[string][]string) // layerID -> list of statuses
	var layerOrder []string                  // maintain layer order

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

		// Track layer and its status
		if msg.ID != "" {
			if _, exists := layerStatus[msg.ID]; !exists {
				layerOrder = append(layerOrder, msg.ID)
				layerStatus[msg.ID] = []string{}
			}

			// Add status if it's new and meaningful
			if msg.Status != "" {
				currentStatuses := layerStatus[msg.ID]
				// Only add if this is a new meaningful status
				if !containsStatus(currentStatuses, msg.Status) {
					layerStatus[msg.ID] = append(currentStatuses, msg.Status)
				}
			}
		}

		// Show digest
		if msg.ID == "" && strings.Contains(msg.Status, "Digest:") {
			fmt.Printf("📦 %s\n", msg.Status)
		}
	}

	// Now display layers one by one in order
	for i, layerID := range layerOrder {
		statuses := layerStatus[layerID]
		fmt.Printf("Layer %d: %s\n", i+1, layerID)

		// Show the progression of this layer
		for _, status := range statuses {
			if strings.Contains(status, "Pushing") {
				fmt.Printf("   📤 Uploading\n")
			} else if strings.Contains(status, "Pushed") {
				fmt.Printf("   ✅ Pushed\n")
			} else if strings.Contains(status, "Layer already exists") {
				fmt.Printf("   ⚡ Already exists\n")
			} else if strings.Contains(status, "Mounted") {
				fmt.Printf("   🔗 Mounted from cache\n")
			} else if strings.Contains(status, "Verifying Checksum") {
				fmt.Printf("   🔍 Verifying\n")
			} else if strings.Contains(status, "Preparing") {
				fmt.Printf("   📦 Preparing\n")
			} else if strings.Contains(status, "Waiting") {
				fmt.Printf("   ⏳ Waiting\n")
			}
		}

		// If no specific status was captured, show completed
		if len(statuses) == 0 {
			fmt.Printf("   ✅ Completed\n")
		}
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

func containsStatus(statuses []string, status string) bool {
	for _, s := range statuses {
		if strings.Contains(s, status) || strings.Contains(status, s) {
			return true
		}
	}
	return false
}
