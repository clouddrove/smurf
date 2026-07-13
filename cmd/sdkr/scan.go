package sdkr

import (
	"errors"
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var scanOutputFormat string

// scanCmd provides functionality to scan a Docker image for known security issues.
// It supports both direct command-line arguments and configuration file values for the image name,
// and optionally allows saving the scan report to a specified SARIF file.
var scanCmd = &cobra.Command{
	Use:          "scan [IMAGE_NAME[:TAG]]",
	Short:        "Scan a Docker image for known vulnerabilities.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !utils.ValidOutputFormat(scanOutputFormat, "table", "json") {
			return fmt.Errorf("invalid output format %q: must be one of table, json", scanOutputFormat)
		}
		isTable := scanOutputFormat == "" || scanOutputFormat == "table"

		var imageRef string
		if len(args) == 1 {
			imageRef = args[0]
		} else {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}
			if data.Sdkr.ImageName == "" {
				if isTable {
					pterm.Error.Printfln("image name (with optional tag) must be provided either as an argument or in the config")
				}
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName
		}

		if isTable {
			pterm.Info.Printf("Scanning Docker image %q...\n", imageRef)
		}
		err := docker.Trivy(imageRef, scanOutputFormat, useAI)
		if err != nil {
			return err
		}

		if isTable {
			pterm.Success.Println("Scan completed successfully.")
		}
		return nil
	},
	Example: `
 smurf sdkr scan my-image:latest
 smurf sdkr scan
 # In the second example, it will read IMAGE_NAME from the config file

 smurf sdkr scan my-image:latest -o json
 # Prints the trivy scan report as a JSON document
`,
}

func init() {
	scanCmd.Flags().StringVarP(&scanOutputFormat, "output", "o", "table", "output format (table|json)")
	scanCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	_ = scanCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json"}, cobra.ShellCompDirectiveDefault
	})

	sdkrCmd.AddCommand(scanCmd)
}
