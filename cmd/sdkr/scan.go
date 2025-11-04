package sdkr

import (
	"errors"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// scanCmd provides functionality to scan a Docker image for known security issues.
// It supports both direct command-line arguments and configuration file values for the image name,
// and optionally allows saving the scan report to a specified SARIF file.
var scanCmd = &cobra.Command{
	Use:          "scan [IMAGE_NAME[:TAG]]",
	Short:        "Scan a Docker image for known vulnerabilities.",
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

		pterm.Info.Printf("Scanning Docker image %q...\n", imageRef)
		err := docker.Trivy(imageRef)
		if err != nil {
			return err
		}

		pterm.Success.Println("Scan completed successfully.")
		return nil
	},
	Example: `
 smurf sdkr scan my-image:latest
 smurf sdkr scan
 # In the second example, it will read IMAGE_NAME from the config file
`,
}

func init() {
	sdkrCmd.AddCommand(scanCmd)
}
