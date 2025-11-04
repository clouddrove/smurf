package sdkr

import (
	"errors"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// removeCmd defines the "remove" command to delete a specified Docker image
// from the local system. If no image is provided, it reads from the config file.
// On successful removal, a confirmation message is displayed.
var removeCmd = &cobra.Command{
	Use:          "remove [IMAGE_NAME[:TAG]]",
	Short:        "Remove a Docker image from the local system.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageRef string

		if len(args) == 1 {
			imageRef = args[0]
		} else {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if data.Sdkr.ImageName == "" {
				pterm.Error.Printfln("image name (with optional tag) must be provided either as an argument or in the config")
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName
		}

		pterm.Info.Printfln("Removing Docker image %v...\n", imageRef)
		err := docker.RemoveImage(imageRef)
		if err != nil {
			return err
		}
		pterm.Success.Println("Image removal completed successfully.")
		return nil
	},
	Example: `
  smurf sdkr remove my-image:latest
  smurf sdkr remove
  # In the second example, it will read IMAGE_NAME from the config file
`,
}

func init() {
	sdkrCmd.AddCommand(removeCmd)
}
