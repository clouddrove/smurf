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
func TestTagImage(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	require.NoError(t, err)
	defer cli.Close()

	_, err = cli.ImagePull(context.Background(), "alpine:latest", image.PullOptions{})
	require.NoError(t, err)

	tests := []struct {
		name    string
		opts    docker.TagOptions
		wantErr bool
	}{
		{
			name: "successful tag",
			opts: docker.TagOptions{
				Source: "alpine:latest",
				Target: "test-tag-image:latest",
			},
			wantErr: false,
		},
		{
			name: "invalid source image",
			opts: docker.TagOptions{
				Source: "nonexistent-image:latest",
				Target: "new-tag:latest",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := docker.TagImage(tt.opts, false)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			_, _, err = cli.ImageInspectWithRaw(context.Background(), tt.opts.Target)
			assert.NoError(t, err)
		})
	}
}
