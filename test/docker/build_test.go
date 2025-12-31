package docker_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clouddrove/smurf/internal/docker"
)

// TestBuild tests the Build function by creating a Dockerfile in a temporary directory,
// building an image from it, and then checking if the image exists.
func TestBuild(t *testing.T) {
	testDir := t.TempDir()

	dockerfile := `FROM alpine:latest
CMD ["echo", "hello"]`

	dockerfilePath := filepath.Join(testDir, "Dockerfile")
	err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644)
	require.NoError(t, err)

	opts := docker.BuildOptions{
		ContextDir:     testDir,
		DockerfilePath: dockerfilePath,
		Timeout:        time.Minute * 5,
	}

	imageName := "test-image"
	tag := "latest"
	err = docker.Build(imageName, tag, opts, false)
	require.NoError(t, err)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err)
	defer cli.Close()

	fullImageName := fmt.Sprintf("%s:%s", imageName, tag)
	_, _, err = cli.ImageInspectWithRaw(context.Background(), fullImageName)
	assert.NoError(t, err)
}
