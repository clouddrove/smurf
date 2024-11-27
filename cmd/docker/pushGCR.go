package docker

import (
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	gcrProjectID       string
	gcrImageName       string
	gcrImageTag        string
	gcrDeleteAfterPush bool
	gcrAuto            bool
)

var pushGcrCmd = &cobra.Command{
	Use:   "gcp",
	Short: "push Docker images to GCR",
	Long: `push Docker images to Google Container Registry
	Set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of your service account JSON key file.
	export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if gcrAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			envVars := map[string]string{
				"GOOGLE_APPLICATION_CREDENTIALS": data.Sdkr.GoogleApplicationCredentials,
			}

			if err := configs.ExportEnvironmentVariables(envVars); err != nil {
				fmt.Println("Error exporting variables:", err)
				return err
			}

			sampleImageNameForGcr := "my-image"

			if gcrProjectID == "" {
				gcrProjectID = data.Sdkr.ProvisionGcrProjectID
			}

			if gcrImageName == "" {
				gcrImageName = sampleImageNameForGcr
			}

		}

		if gcrProjectID == "" {
			return fmt.Errorf("gcp requires --project-id flag")
		}

		gcrImage := fmt.Sprintf("gcr.io/%s/%s:%s", gcrProjectID, gcrImageName, gcrImageTag)

		pterm.Info.Println("Pushing image to Google Container Registry...")
		if err := docker.PushImageToGCR(gcrProjectID, gcrImageName); err != nil {
			return err
		}
		pterm.Success.Println("Successfully pushed image to GCR:", gcrImage)

		if gcrDeleteAfterPush {
			if err := docker.RemoveImage(gcrImageName); err != nil {
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", gcrImageName)
		}

		return nil
	},
	Example: `
	smurf sdkr push gcp --project-id <project-id> --image <image-name> --tag <image-tag>
	smurf sdkr push gcp --project-id <project-id> --image <image-name> --tag <image-tag> --delete
	`,
}

func init() {
	pushGcrCmd.Flags().StringVarP(&gcrImageName, "image", "i", "", "Image name (e.g., myapp)")
	pushGcrCmd.Flags().StringVarP(&gcrImageTag, "tag", "t", "latest", "Image tag (default: latest)")
	pushGcrCmd.Flags().BoolVarP(&gcrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	pushGcrCmd.Flags().BoolVarP(&gcrAuto, "auto", "a", false, "Use the default image name and tag from the config file")
	pushGcrCmd.Flags().StringVar(&gcrProjectID, "project-id", "", "GCP project ID (required with --gcp)")

	pushGcrCmd.MarkFlagRequired("project-id")
	pushGcrCmd.MarkFlagRequired("image")

	pushCmd.AddCommand(pushGcrCmd)
}
