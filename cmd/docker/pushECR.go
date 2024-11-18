package docker

import (
	"fmt"

	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	ecrImageName      string
	ecrRepositoryName string
	ecrRegionName     string
	ecrImageTag   string
	ecrDeleteAfterPush bool
)

var pushEcrCmd = &cobra.Command{
	Use:   "aws",
	Short: "push Docker images to ECR",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ecrRegionName == "" || ecrRepositoryName == "" {
			return fmt.Errorf("aws requires both --region and --repository flags")
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
}

func init() {
	pushEcrCmd.Flags().StringVarP(&ecrImageName, "image", "i", "", "Image name (e.g., myapp)")
	pushEcrCmd.Flags().StringVarP(&ecrImageTag, "tag", "t", "latest", "Image tag (default: latest)")
	pushEcrCmd.Flags().BoolVarP(&ecrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	pushEcrCmd.Flags().StringVarP(&ecrRegionName, "region", "r", "", "AWS region (required with --aws)")
	pushEcrCmd.Flags().StringVarP(&ecrRepositoryName, "repository", "R", "", "AWS ECR repository name (required with --aws)")

	pushEcrCmd.MarkFlagRequired("region")
	pushEcrCmd.MarkFlagRequired("repository")
	pushEcrCmd.MarkFlagRequired("image")

	pushCmd.AddCommand(pushEcrCmd)
}
