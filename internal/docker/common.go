package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// encodeAuthToBase64 encodes the given registry.AuthConfig as a base64-encoded string.
// used in the push function to authenticate with the Docker registry.
func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

func initDockerClient(timeout time.Duration) (*client.Client, context.Context, context.CancelFunc, error) {
	fmt.Printf("Initializing Docker client...\n")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("failed to initialize Docker client: %w", err)
	}
	return cli, ctx, cancel, nil
}

func prepareAuth(username, password, server string) (string, error) {
	authConfig := registry.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: server,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("failed to encode auth: %w", err)
	}
	return base64.URLEncoding.EncodeToString(encodedJSON), nil
}

// decodePushStream reads the newline-delimited JSON stream produced by the
// Docker push API. It aborts on the first error message carried in the
// stream and on any stream decode failure other than a clean EOF, since both
// cases mean the push cannot be trusted to have completed successfully.
func decodePushStream(r io.Reader) (layerOrder []string, layerStatus map[string][]string, err error) {
	decoder := json.NewDecoder(r)
	layerStatus = make(map[string][]string)

	for {
		var msg jsonMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return layerOrder, layerStatus, fmt.Errorf("decoding push response: %w", err)
		}
		if msg.Error != "" {
			return layerOrder, layerStatus, fmt.Errorf("push error: %s", msg.Error)
		}

		if msg.ID != "" {
			if _, exists := layerStatus[msg.ID]; !exists {
				layerOrder = append(layerOrder, msg.ID)
				layerStatus[msg.ID] = []string{}
			}

			if msg.Status != "" && isMeaningfulStatus(msg.Status) && !hasSimilarStatus(layerStatus[msg.ID], msg.Status) {
				layerStatus[msg.ID] = append(layerStatus[msg.ID], msg.Status)
			}
		}

		if msg.ID == "" && strings.Contains(msg.Status, "Digest:") {
			fmt.Printf("📦 %s\n", msg.Status)
		}
	}

	return layerOrder, layerStatus, nil
}

// Core push logic shared between GHCR and other registries
func pushImage(cli *client.Client, ctx context.Context, imageName, authStr string) error {
	fmt.Printf("Pushing image: %s\n", imageName)
	fmt.Println("─────────────────────────────────────────────────────────────")

	pushResp, err := cli.ImagePush(ctx, imageName, image.PushOptions{RegistryAuth: authStr})
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer pushResp.Close()

	layerOrder, layerStatus, err := decodePushStream(pushResp)
	if err != nil {
		return err
	}

	displayLayerProgress(layerOrder, layerStatus)

	fmt.Println("─────────────────────────────────────────────────────────────")
	fmt.Printf("✅ Successfully pushed image: %s\n", imageName)
	fmt.Printf("📦 Total layers processed: %d\n", len(layerOrder))
	return nil
}

// Displays formatted layer progression
func displayLayerProgress(layerOrder []string, layerStatus map[string][]string) {
	for i, layerID := range layerOrder {
		fmt.Printf("Layer %d: %s\n", i+1, layerID)
		for _, status := range layerStatus[layerID] {
			switch {
			case strings.Contains(status, "Preparing"):
				fmt.Println("   📦 Preparing")
			case strings.Contains(status, "Waiting"):
				fmt.Println("   ⏳ Waiting")
			case strings.Contains(status, "Layer already exists"):
				fmt.Println("   ✅ Already exists")
			case strings.Contains(status, "Pushing"):
				fmt.Println("   📤 Uploading")
			case strings.Contains(status, "Pushed"):
				fmt.Println("   ✅ Pushed")
			case strings.Contains(status, "Mounted"):
				fmt.Println("   🔗 Mounted from cache")
			case strings.Contains(status, "Verifying Checksum"):
				fmt.Println("   🔍 Verifying")
			}
		}
	}
}
