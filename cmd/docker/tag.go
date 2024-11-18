package docker

import (
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/spf13/cobra"
)

var sourceTag string
var targetTag string

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag a Docker image for a remote repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := docker.TagOptions{
			Source: sourceTag,
			Target: targetTag,
		}
		return docker.TagImage(opts)
	},
}

func init() {
	tagCmd.Flags().StringVarP(&sourceTag, "source", "s", "", "Source image tag (format: image:tag)")
	tagCmd.Flags().StringVarP(&targetTag, "target", "t", "", "Target image tag (format: repository/image:tag)")
	tagCmd.MarkFlagRequired("source")
	tagCmd.MarkFlagRequired("target")

	sdkrCmd.AddCommand(tagCmd)
}
