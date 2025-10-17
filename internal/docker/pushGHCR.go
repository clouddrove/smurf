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
	"github.com/pterm/pterm"
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
	spinner, _ := pterm.DefaultSpinner.Start("Initializing Docker client for GHCR...")

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		spinner.Fail("Failed to initialize Docker client\n")
		return fmt.Errorf("failed to initialize Docker client : %v", err)
	}
	defer cli.Close()

	spinner.UpdateText("Preparing GHCR authentication...")

	// Use GitHub credentials for GHCR
	authConfig := registry.AuthConfig{
		Username:      os.Getenv("GITHUB_USERNAME"),
		Password:      os.Getenv("GITHUB_TOKEN"),
		ServerAddress: "ghcr.io",
	}

	// Validate that we have GitHub credentials
	if authConfig.Username == "" || authConfig.Password == "" {
		spinner.Fail("GitHub credentials not found")
		return fmt.Errorf("GITHUB_USERNAME and GITHUB_TOKEN environment variables are required for GHCR")
	}

	// Validate that image name is for GHCR
	if !strings.HasPrefix(opts.ImageName, "ghcr.io/") {
		spinner.Fail("Image name must be for GHCR registry")
		return fmt.Errorf("image name must start with 'ghcr.io/' for GitHub Container Registry")
	}

	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		spinner.Fail("GHCR authentication preparation failed: ", err)
		return fmt.Errorf("GHCR authentication preparation failed : %v", err)
	}

	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	spinner.UpdateText(fmt.Sprintf("Pushing image %s to GHCR...", opts.ImageName))

	pushResp, err := cli.ImagePush(ctx, opts.ImageName, image.PushOptions{
		RegistryAuth: authStr,
	})

	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to push image %s to GHCR, error : %v", opts.ImageName, err))
		return fmt.Errorf("failed to push image to GHCR : %v", err)
	}
	defer pushResp.Close()

	decoder := json.NewDecoder(pushResp)
	progressBar, _ := pterm.DefaultProgressbar.WithTotal(100).WithTitle("GHCR Push Progress").Start()

	for {
		var msg jsonMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			// Don't fail on decode errors, just break
			break
		}

		if msg.Error != "" {
			spinner.Fail(msg.Error)
			return fmt.Errorf("failed to push image to GHCR : %v", msg.Error)
		}

		if msg.Status != "" {
			// Update progress based on status messages
			if strings.Contains(msg.Status, "Layer already exists") {
				progressBar.Add(10)
			} else if strings.Contains(msg.Status, "Pushed") {
				progressBar.Add(15)
			} else if strings.Contains(msg.Status, "Mounted") {
				progressBar.Add(5)
			}

			spinner.UpdateText(msg.Status)
		}

		if msg.Progress != "" {
			progressBar.UpdateTitle(fmt.Sprintf("%s: %s", msg.Status, msg.Progress))
		}
	}

	progressBar.Stop()
	spinner.Success("Successfully pushed image to GHCR: ", opts.ImageName)

	return nil
}
