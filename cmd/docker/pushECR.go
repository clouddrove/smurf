package docker

import (
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	ecrImageName       string
	ecrRepositoryName  string
	ecrRegionName      string
	ecrImageTag        string
	ecrDeleteAfterPush bool
	ecrAuto            bool
)

var pushEcrCmd = &cobra.Command{
	Use:   "aws",
	Short: "push Docker images to ECR",
	RunE: func(cmd *cobra.Command, args []string) error {

		if ecrAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			sampleImageNameForEcr := "my-image"

			if ecrRegionName == "" {
				ecrRegionName = data.Sdkr.ProvisionEcrRegion
			}

			if ecrRepositoryName == "" {
				ecrRepositoryName = data.Sdkr.ProvisionEcrRepository
			}

			if ecrImageName == "" {
				ecrImageName = sampleImageNameForEcr
			}
		}

		if ecrRegionName == "" || ecrRepositoryName == "" || ecrImageName == "" {
			pterm.Error.Println("Required flags are missing. Please provide the required flags.")
		}

		ecrImage := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s", ecrImageName, ecrRegionName, ecrRepositoryName, ecrImageTag)
		pterm.Info.Println("Pushing image to AWS ECR...")
		if err := docker.PushImageToECR(ecrImageName, ecrRegionName, ecrRepositoryName); err != nil {
			return err
		}
		pterm.Success.Println("Successfully pushed image to ECR:", ecrImage)

		if ecrDeleteAfterPush {
			if err := docker.RemoveImage(ecrImageName); err != nil {
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", ecrImageName)
		}
		return nil
	},
	Example: `
	smurf sdkr push aws --region <region> --repository <repository> --image <image-name> --tag <image-tag>
	smurf sdkr push aws --region <region> --repository <repository> --image <image-name> --tag <image-tag> --delete
	`,
}

func init() {
	pushEcrCmd.Flags().StringVarP(&ecrImageName, "image", "i", "", "Image name (e.g., myapp)")
	pushEcrCmd.Flags().StringVarP(&ecrImageTag, "tag", "t", "latest", "Image tag (default: latest)")
	pushEcrCmd.Flags().BoolVarP(&ecrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	pushEcrCmd.Flags().BoolVarP(&ecrAuto, "auto", "a", false, "Use the default image name and tag from the config file")
	pushEcrCmd.Flags().StringVarP(&ecrRegionName, "region", "r", "", "AWS region (required with --aws)")
	pushEcrCmd.Flags().StringVarP(&ecrRepositoryName, "repository", "R", "", "AWS ECR repository name (required with --aws)")


	pushCmd.AddCommand(pushEcrCmd)
}
