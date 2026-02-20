package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
)

// Push to GitHub Container Registry (GHCR)
func PushToGHCR(opts PushOptions, useAI bool) error {
	if !strings.HasPrefix(opts.ImageName, "ghcr.io/") {
		return fmt.Errorf("image name must start with 'ghcr.io/' for GHCR")
	}

	cli, ctx, cancel, err := initDockerClient(opts.Timeout)
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return err
	}
	defer cancel()
	defer cli.Close()

	fmt.Printf("Preparing GHCR authentication...\n")
	authStr, err := prepareAuth(os.Getenv("GITHUB_USERNAME"), os.Getenv("GITHUB_TOKEN"), "ghcr.io")
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	if os.Getenv("GITHUB_USERNAME") == "" || os.Getenv("GITHUB_TOKEN") == "" {
		return fmt.Errorf("GITHUB_USERNAME and GITHUB_TOKEN environment variables are required for GHCR")
	}

	err = pushImage(cli, ctx, opts.ImageName, authStr)
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
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
