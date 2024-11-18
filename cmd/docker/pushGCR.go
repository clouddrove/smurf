package docker

import (
	"fmt"

	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	gcrProjectID       string
	gcrImageName       string
	gcrImageTag        string
	gcrDeleteAfterPush bool
)

var pushGcrCmd = &cobra.Command{
	Use:   "gcp",
	Short: "push Docker images to GCR",
	Long: `push Docker images to Google Container Registry
	Set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of your service account JSON key file.
	export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
}

func init() {
	pushGcrCmd.Flags().StringVarP(&gcrImageName, "image", "i", "", "Image name (e.g., myapp)")
	pushGcrCmd.Flags().StringVarP(&gcrImageTag, "tag", "t", "latest", "Image tag (default: latest)")
	pushGcrCmd.Flags().BoolVarP(&gcrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	pushGcrCmd.Flags().StringVar(&gcrProjectID, "project-id", "", "GCP project ID (required with --gcp)")

	pushGcrCmd.MarkFlagRequired("project-id")
	pushGcrCmd.MarkFlagRequired("image")

	pushCmd.AddCommand(pushGcrCmd)
}
