package docker

import (
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var imageTag string
var local bool
var hub bool

var remove = &cobra.Command{
	Use:   "remove",
	Short: "Remove Docker images",
	RunE: func(cmd *cobra.Command, args []string) error {

		err := docker.RemoveImage(imageTag)
		if err != nil {
			pterm.Error.Println(err)
			return err
		}
		pterm.Success.Println("Image removal completed successfully.")
		return nil
	},
}

func init() {
	remove.Flags().StringVarP(&imageTag, "tag", "t", "", "Docker image tag to remove")
	remove.MarkFlagRequired("tag")

	sdkrCmd.AddCommand(remove)
}
