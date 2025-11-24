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

// pushGcrCmd defines the "gcp" subcommand for pushing Docker images to Google Container Registry or Artifact Registry.
var pushGcrCmd = &cobra.Command{
	Use:   "gcp [IMAGE_NAME[:TAG]]",
	Short: "Push Docker images to Google Container Registry or Artifact Registry",
	Long: `Push Docker images to Google Container Registry or Artifact Registry. 

Authentication Methods:
1. gcloud CLI (recommended): Run 'gcloud auth login' and 'gcloud auth configure-docker'
2. Service Account: Set GOOGLE_APPLICATION_CREDENTIALS environment variable

Supports:
- GCR: gcr.io/PROJECT_ID/IMAGE_NAME:TAG
- Artifact Registry: REGION-docker.pkg.dev/PROJECT_ID/REPOSITORY/IMAGE_NAME:TAG
- Short form: IMAGE_NAME:TAG (automatically uses Artifact Registry)
`,
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
			if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" && data.Sdkr.GoogleApplicationCredentials != "" {
				envVars = map[string]string{
					"GOOGLE_APPLICATION_CREDENTIALS": data.Sdkr.GoogleApplicationCredentials,
				}
			}
			if envVars != nil {
				if err := configs.ExportEnvironmentVariables(envVars); err != nil {
					return err
				}
			}
		}

		// Verify authentication before proceeding
		pterm.Info.Println("Verifying Google Cloud authentication...")
		if err := docker.VerifyGCloudAuth(); err != nil {
			pterm.Error.Printf("Authentication verification failed: %v\n", err)
			return err
		}

		// Determine registry type for better messaging
		registryType := "Google Container Registry"
		if strings.Contains(imageRef, ".pkg.dev") {
			registryType = "Artifact Registry"
		} else if !strings.Contains(imageRef, "gcr.io") && configs.ProjectID != "" {
			// If it's a short name and we have project ID, we'll use Artifact Registry
			registryType = "Artifact Registry"
		}

		pterm.Info.Printf("Pushing image to %s...\n", registryType)

		// Pass the full image reference to PushImageToGCR
		if err := docker.PushImageToGCR(configs.ProjectID, imageRef); err != nil {
			pterm.Error.Printf("Failed to push image to %s: %v\n", registryType, err)
			return err
		}

		// Construct a success message after the push is successful
		pterm.Success.Printf("Successfully pushed image to %s: %s\n", registryType, imageRef)

		if configs.DeleteAfterPush {
			// Extract base image name for deletion
			baseName := imageRef
			if parts := strings.Split(imageRef, ":"); len(parts) > 0 {
				baseName = parts[0]
			}

			pterm.Info.Printf("Deleting local image %s...\n", baseName)
			if err := docker.RemoveImage(baseName); err != nil {
				pterm.Warning.Printf("Failed to delete local image %s: %v\n", baseName, err)
				// Don't return error here, as the push was successful
			} else {
				pterm.Success.Println("Successfully deleted local image:", baseName)
			}
		}

		return nil
	},
	Example: `  # Push to Artifact Registry with full image name
  smurf sdkr push gcp us-central1-docker.pkg.dev/my-project/my-repo/myapp:v1

  # Push to GCR with full image name
  smurf sdkr push gcp gcr.io/my-project/myapp:v1

  # Push with short name (uses Artifact Registry)
  smurf sdkr push gcp myapp:v1 --project-id my-project

  # Push and delete local image
  smurf sdkr push gcp myapp:v1 --project-id my-project --delete`,
}

func init() {
	pushGcrCmd.Flags().StringVar(&configs.ProjectID, "project-id", "", "GCP project ID (required for short image names)")
	pushGcrCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	pushCmd.AddCommand(pushGcrCmd)
}
