package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// PushImage pushes the specified Docker image to the Docker Hub.
// It authenticates with Docker Hub, tags the image, and pushes it to the registry.
// It displays a spinner with progress updates and prints the push response messages.
func PushImage(opts PushOptions) error {
	fmt.Printf("Initializing Docker client...\n")

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("❌ Failed to initialize Docker client\n")
		return fmt.Errorf("failed to initialize Docker client: %v", err)
	}
	defer cli.Close()

	fmt.Printf("Preparing authentication...\n")

	authConfig := registry.AuthConfig{
		Username: os.Getenv("DOCKER_USERNAME"),
		Password: os.Getenv("DOCKER_PASSWORD"),
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		fmt.Printf("❌ Authentication preparation failed: %v\n", err)
		return fmt.Errorf("authentication preparation failed: %v", err)
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	fmt.Printf("Pushing image: %s\n", opts.ImageName)
	fmt.Printf("─────────────────────────────────────────────────────────────\n")

	pushResp, err := cli.ImagePush(ctx, opts.ImageName, image.PushOptions{
		RegistryAuth: authStr,
	})

	if err != nil {
		fmt.Printf("❌ Failed to push image %s, error: %v\n", opts.ImageName, err)
		return fmt.Errorf("failed to push image: %v", err)
	}
	defer pushResp.Close()

	decoder := json.NewDecoder(pushResp)
	var layerCount int
	var currentLayerID string

	for {
		var msg jsonMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if msg.Error != "" {
			fmt.Printf("❌ Push error: %s\n", msg.Error)
			return fmt.Errorf("failed to push image: %v", msg.Error)
		}

		// Handle layer identification
		if msg.ID != "" && msg.ID != currentLayerID {
			currentLayerID = msg.ID
			layerCount++
			if msg.ID != "" {
				fmt.Printf("Layer %d: %s\n", layerCount, msg.ID)
			}
		}

		// Display progress information
		if msg.Status != "" {
			switch {
			case strings.Contains(msg.Status, "Preparing"):
				fmt.Printf("   📦 %s\n", msg.Status)
			case strings.Contains(msg.Status, "Waiting"):
				fmt.Printf("   ⏳ %s\n", msg.Status)
			case strings.Contains(msg.Status, "Layer already exists"):
				fmt.Printf("   ✅ %s\n", msg.Status)
			case strings.Contains(msg.Status, "Pushing") && !strings.Contains(msg.Progress, "MB"):
				// Only show "Pushing" once per layer, not the progress updates
				fmt.Printf("   📤 %s\n", msg.Status)
			case strings.Contains(msg.Status, "Pushed"):
				fmt.Printf("   ✅ %s\n", msg.Status)
			case strings.Contains(msg.Status, "Mounted"):
				fmt.Printf("   🔗 %s\n", msg.Status)
			case strings.Contains(msg.Status, "Verifying Checksum"):
				fmt.Printf("   🔍 %s\n", msg.Status)
			case strings.Contains(msg.Status, "Digest:"):
				fmt.Printf("   🏷️  %s\n", msg.Status)
			}
		}
	}

	fmt.Printf("─────────────────────────────────────────────────────────────\n")
	fmt.Printf("✅ Successfully pushed image: %s\n", opts.ImageName)
	fmt.Printf("📦 Total layers processed: %d\n", layerCount)

	return nil
}
