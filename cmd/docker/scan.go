package docker

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	dockerTag string
	sarifFile string
	scanAuto  bool
)

var scan = &cobra.Command{
	Use:   "scan",
	Short: "Scan Docker images for known vulnerabilities",
	RunE: func(cmd *cobra.Command, args []string) error {

		if scanAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if dockerTag == "" {
				dockerTag = data.Sdkr.SourceTag
			}

			if sarifFile == "" {
				sarifFile = "scan.json"
			}
		}

		if dockerTag == "" {
			pterm.Error.Println("Required flags are missing. Please provide the required flags.")
		}

		err := docker.Scout(dockerTag, sarifFile)
		if err != nil {
			pterm.Error.Println(err)
			return err
		}
		return nil
	},
	Example: `
    smurf sdkr scan --tag <image-name> --output <sarif-file>
    `,
}

func init() {
	scan.Flags().StringVarP(&dockerTag, "tag", "t", "", "Docker image tag to scan")
	scan.Flags().StringVarP(&sarifFile, "output", "o", "", "Output file for SARIF report")
	scan.Flags().BoolVarP(&scanAuto, "auto", "a", false, "Scan Docker image automatically")

	sdkrCmd.AddCommand(scan)
}
