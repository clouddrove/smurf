//go:build integration

package docker_test

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clouddrove/smurf/internal/docker"
)

// TestRemoveImage tests the RemoveImage function by pulling an image, removing it, and then checking if it still exists.
// It also tests the case where the image does not exist.
func TestRemoveImage(t *testing.T) {
	// The daemon on CI runners can be older than the API version this SDK
	// defaults to, so negotiate down to what it supports. internal/docker does
	// the same on every client it builds; without it these helper clients fail
	// with "client version X is too new" while the code under test succeeds.
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	defer cli.Close()

	_, err = cli.ImagePull(context.Background(), "alpine:latest", image.PullOptions{})
	require.NoError(t, err)

	t.Run("successful removal", func(t *testing.T) {
		err := docker.RemoveImage("alpine:latest", false)
		assert.NoError(t, err)

		_, _, err = cli.ImageInspectWithRaw(context.Background(), "alpine:latest")
		assert.Error(t, err, "Image should not exist after removal")
	})

	t.Run("non-existent image", func(t *testing.T) {
		err := docker.RemoveImage("nonexistent-image:latest", false)
		assert.Error(t, err)
	})
}
