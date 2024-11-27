package docker

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/spf13/cobra"
)

var (
	sourceTag string
	targetTag string
	tagAuto   bool
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Tag a Docker image for a remote repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		if tagAuto && (sourceTag != "") && (targetTag != "") {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if sourceTag == "" {
				sourceTag = data.Sdkr.SourceTag
			}

			if targetTag == "" {
				targetTag = data.Sdkr.TargetTag
			}
		}
		opts := docker.TagOptions{
			Source: sourceTag,
			Target: targetTag,
		}
		return docker.TagImage(opts)
	},
	Example: `
	smurf sdkr tag --source <image:tag> --target <repository/image:tag>
	`,
}

func init() {
	tagCmd.Flags().StringVarP(&sourceTag, "source", "s", "", "Source image tag (format: image:tag)")
	tagCmd.Flags().StringVarP(&targetTag, "target", "t", "", "Target image tag (format: repository/image:tag)")
	tagCmd.Flags().BoolVarP(&tagAuto, "auto", "a", false, "Tag Docker image automatically")
	tagCmd.MarkFlagRequired("source")
	tagCmd.MarkFlagRequired("target")

	sdkrCmd.AddCommand(tagCmd)
}
