package sdkr

import (
	"errors"
	"fmt"
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/spf13/cobra"
	"os"
)

// scanCmd provides functionality to scan a Docker image for known security issues.
// It supports both direct command-line arguments and configuration file values for the image name,
// and optionally allows saving the scan report to a specified SARIF file.
var scanCmd = &cobra.Command{
	Use:   "scan [IMAGE_NAME[:TAG]]",
	Short: "Scan a Docker image for known vulnerabilities.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageRef string
		var sarifPath string

		// Check if image is passed as argument or read from config
		if len(args) == 1 {
			imageRef = args[0]
		} else {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if data.Sdkr.ImageName == "" {
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName
			sarifPath = data.Sdkr.SarifPath // Optional SARIF output path from config
		}

		// Plain log without spinner
		fmt.Printf("🔍 Scanning Docker image %q...\n", imageRef)

		// Run Trivy scan
		err := docker.Trivy(imageRef, sarifPath)
		if err != nil {
			fmt.Printf("❌ Scan failed: %v\n", err)
			return err
		}

		// Success message without spinner
		fmt.Println("✅ Scan completed successfully.")
		return nil
	},
	Example: `
 smurf sdkr scan my-image:latest
 smurf sdkr scan
 # In the second example, it will read IMAGE_NAME from the config file
`,
}

// init adds the scan command to the parent sdkr command.
func init() {
	sdkrCmd.AddCommand(scanCmd)
}
