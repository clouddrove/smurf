package docker

import (
	"fmt"
	"os"
	"strings"
)

// Push to GitHub Container Registry (GHCR)
func PushToGHCR(opts PushOptions) error {
	if !strings.HasPrefix(opts.ImageName, "ghcr.io/") {
		return fmt.Errorf("image name must start with 'ghcr.io/' for GHCR")
	}

	cli, ctx, cancel, err := initDockerClient(opts.Timeout)
	if err != nil {
		return err
	}
	defer cancel()
	defer cli.Close()

	fmt.Printf("Preparing GHCR authentication...\n")
	authStr, err := prepareAuth(os.Getenv("USERNAME_GITHUB"), os.Getenv("TOKEN_GITHUB"), "ghcr.io")
	if err != nil {
		return err
	}

	if os.Getenv("USERNAME_GITHUB") == "" || os.Getenv("TOKEN_GITHUB") == "" {
		return fmt.Errorf("USERNAME_GITHUB and TOKEN_GITHUB environment variables are required for GHCR")
	}

	err = pushImage(cli, ctx, opts.ImageName, authStr)
	if err != nil {
		return err
	}

	parts := strings.Split(opts.ImageName, "/")
	if len(parts) >= 3 {
		repoParts := strings.Split(parts[2], ":")
		repoName := repoParts[0]
		fmt.Printf("🌐 View at: https://github.com/%s/pkgs/container/%s\n", parts[1], repoName)
	}
	return nil
}
