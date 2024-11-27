package docker

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	imageTag   string
	local      bool
	hub        bool
	removeAuto bool
)

var remove = &cobra.Command{
	Use:   "remove",
	Short: "Remove Docker images",
	RunE: func(cmd *cobra.Command, args []string) error {

		if removeAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			imageTag = data.Sdkr.SourceTag
		}

		err := docker.RemoveImage(imageTag)
		if err != nil {
			pterm.Error.Println(err)
			return err
		}
		pterm.Success.Println("Image removal completed successfully.")
		return nil
	},
	Example: `
	smurf sdkr remove --tag <image-name>
	`,
}

func init() {
	remove.Flags().StringVarP(&imageTag, "tag", "t", "", "Docker image tag to remove")
	remove.Flags().BoolVarP(&removeAuto, "auto", "a", false, "Remove Docker image automatically")
	remove.MarkFlagRequired("tag")

	sdkrCmd.AddCommand(remove)
}
