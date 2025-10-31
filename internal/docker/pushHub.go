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
			fmt.Printf("❌ Push error: %s\n", msg.Error)
			return fmt.Errorf("failed to push image: %v", msg.Error)
		}

		// Track layer and its status
		if msg.ID != "" {
			if _, exists := layerStatus[msg.ID]; !exists {
				layerOrder = append(layerOrder, msg.ID)
				layerStatus[msg.ID] = []string{}
			}

			// Add status if it's meaningful and not already tracked
			if msg.Status != "" && isMeaningfulStatus(msg.Status) {
				currentStatuses := layerStatus[msg.ID]
				if !hasSimilarStatus(currentStatuses, msg.Status) {
					layerStatus[msg.ID] = append(currentStatuses, msg.Status)
				}
			}
		}

		// Show digest
		if msg.ID == "" && strings.Contains(msg.Status, "Digest:") {
			fmt.Printf("📦 %s\n", msg.Status)
		}
	}

	// Display layers one by one in order
	for i, layerID := range layerOrder {
		statuses := layerStatus[layerID]
		fmt.Printf("Layer %d: %s\n", i+1, layerID)

		// Show the progression of this layer
		for _, status := range statuses {
			if strings.Contains(status, "Preparing") {
				fmt.Printf("   📦 Preparing\n")
			} else if strings.Contains(status, "Waiting") {
				fmt.Printf("   ⏳ Waiting\n")
			} else if strings.Contains(status, "Layer already exists") {
				fmt.Printf("   ✅ Already exists\n")
			} else if strings.Contains(status, "Pushing") {
				fmt.Printf("   📤 Uploading\n")
			} else if strings.Contains(status, "Pushed") {
				fmt.Printf("   ✅ Pushed\n")
			} else if strings.Contains(status, "Mounted") {
				fmt.Printf("   🔗 Mounted from cache\n")
			} else if strings.Contains(status, "Verifying Checksum") {
				fmt.Printf("   🔍 Verifying\n")
			}
		}
	}

	fmt.Printf("─────────────────────────────────────────────────────────────\n")
	fmt.Printf("✅ Successfully pushed image: %s\n", opts.ImageName)
	fmt.Printf("📦 Total layers processed: %d\n", len(layerOrder))

	return nil
}

// Helper functions
func isMeaningfulStatus(status string) bool {
	// Filter out noisy status messages
	meaningless := []string{"Image", "latest", "Mounted from"}
	for _, m := range meaningless {
		if strings.Contains(status, m) {
			return false
		}
	}
	return true
}

func hasSimilarStatus(statuses []string, newStatus string) bool {
	for _, status := range statuses {
		if strings.Contains(newStatus, status) || strings.Contains(status, newStatus) {
			return true
		}
	}
	return false
}
