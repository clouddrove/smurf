package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/clouddrove/smurf/internal/ai"
)

// PushImage pushes the specified Docker image to the Docker Hub.
// It authenticates with Docker Hub, tags the image, and pushes it to the registry.
// It displays a spinner with progress updates and prints the push response messages.
func PushImage(opts PushOptions, useAI bool) error {
	cli, ctx, cancel, err := initDockerClient(opts.Timeout)
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return err
	}
	defer cancel()
	defer cli.Close()

	fmt.Printf("Preparing authentication...\n")
	authStr, err := prepareAuth(os.Getenv("DOCKER_USERNAME"), os.Getenv("DOCKER_PASSWORD"), "")
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	return pushImage(cli, ctx, opts.ImageName, authStr)
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
