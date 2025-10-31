package sdkr

import (
	"errors"
	"os"
	"strings"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// pushGcrCmd defines the "gcp" subcommand for pushing Docker images to Google Container Registry (GCR).
// It reads image references from either the command line or config files, supports environment
// variables for authentication, and optionally deletes the local image after a successful push.
var pushGcrCmd = &cobra.Command{
	Use:   "gcp [IMAGE_NAME[:TAG]]",
	Short: "Push Docker images to GCR",
	Long: `Push Docker images to Google Container Registry. Set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of your service account JSON key file, for example:
  export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageRef string
		var envVars map[string]string

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

			if configs.ProjectID == "" {
				configs.ProjectID = data.Sdkr.ProvisionGcrProjectID
			}
			if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
				envVars = map[string]string{
					"GOOGLE_APPLICATION_CREDENTIALS": data.Sdkr.GoogleApplicationCredentials,
				}
			}
			if envVars != nil {
				if err := configs.ExportEnvironmentVariables(envVars); err != nil {
					return err
				}
			}

			if envVars["GOOGLE_APPLICATION_CREDENTIALS"] == "" {
				pterm.Error.Println("Google Application Credentials is required")
				return errors.New("google Application Credentials is required")
			}
		}

		// Don't parse the image name here, let PushImageToGCR handle it
		if configs.ProjectID == "" {
			pterm.Error.Println("GCP project ID is required.")
			return errors.New("GCP project ID is required.")
		}

		pterm.Info.Println("Pushing image to Google Container Registry...")

		// Pass the full image reference to PushImageToGCR
		if err := docker.PushImageToGCR(configs.ProjectID, imageRef); err != nil {
			pterm.Error.Println("Failed to push image to GCR:", err)
			return err
		}

		// Construct a success message after the push is successful
		pterm.Success.Println("Successfully pushed image to GCR:", imageRef)

		if configs.DeleteAfterPush {
			// Extract base image name for deletion
			baseName := imageRef
			if parts := strings.Split(imageRef, ":"); len(parts) > 0 {
				baseName = parts[0]
			}

			pterm.Info.Printf("Deleting local image %s...\n", baseName)
			if err := docker.RemoveImage(baseName); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", baseName)
		}

		return nil
	},
	Example: `  smurf sdkr push gcp myapp:v1 --project-id <project-id>
  smurf sdkr push gcp myapp:v1 --project-id <project-id> --delete`,
}

func init() {
	pushGcrCmd.Flags().StringVar(&configs.ProjectID, "project-id", "", "GCP project ID (required)")
	pushGcrCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	pushCmd.AddCommand(pushGcrCmd)
}
