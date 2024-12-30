package sdkr

import (
	"errors"
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// pushEcrCmd defines the "aws" subcommand for pushing Docker images to AWS ECR.
// It supports reading image references and ECR parameters from either command-line
// arguments or a config file, with an optional cleanup of local images after push.
var pushEcrCmd = &cobra.Command{
	Use:   "aws [IMAGE_NAME[:TAG]]",
	Short: "Push Docker images to ECR",
	Args:  cobra.MaximumNArgs(1),
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
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName

			if configs.Region == "" {
				configs.Region = data.Sdkr.ProvisionEcrRegion
			}
			if configs.Repository == "" {
				configs.Repository = data.Sdkr.ProvisionEcrRepository
			}
		}

		repoName, tag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			return fmt.Errorf("invalid image format: %w", parseErr)
		}
		if repoName == "" {
			return errors.New("invalid image reference")
		}
		if tag == "" {
			tag = "latest"
		}

		if configs.Region == "" || configs.Repository == "" {
			pterm.Error.Println("Required flags are missing. Please provide the required flags.")
			return errors.New("missing required ECR parameters")
		}

		fullEcrImage := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s", repoName, configs.Region, configs.RegistryName, tag)
		pterm.Info.Println("Pushing image to AWS ECR...")

		if err := docker.PushImageToECR(repoName, configs.Region, configs.RegistryName); err != nil {
			pterm.Error.Println("Failed to push image to ECR:", err)
			return err
		}
		pterm.Success.Println("Successfully pushed image to ECR:", fullEcrImage)

		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", repoName)
			if err := docker.RemoveImage(repoName); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", repoName)
		}

		return nil
	},
	Example: `
  smurf sdkr push aws myapp:v1 --region <region> --repository <repository>
  smurf sdkr push aws myapp:v1 --region <region> --repository <repository> --delete
`,
}

func init() {
	pushEcrCmd.Flags().StringVarP(&configs.Region, "region", "r", "", "AWS region (required)")
	pushEcrCmd.Flags().StringVarP(&configs.Repository, "repository", "R", "", "AWS ECR repository name (required)")
	pushEcrCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	pushCmd.AddCommand(pushEcrCmd)
}
