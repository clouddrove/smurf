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
	Use:          "aws [IMAGE_NAME]",
	Short:        "Push Docker images to ECR",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
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
				pterm.Error.Printfln("image name (with optional tag) must be provided either as an argument or in the config")
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName
		}

		accountID, ecrRegionName, ecrRepositoryName, ecrImageTag, parseErr := configs.ParseEcrImageRef(imageRef)
		if parseErr != nil {
			return parseErr
		}

		if accountID == "" || ecrRegionName == "" || ecrRepositoryName == "" || ecrImageTag == "" {
			pterm.Error.Printfln("invalid image reference: missing account ID, region, or repository name")
			return errors.New("invalid image reference: missing account ID, region, or repository name")
		}

		ecrImage := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
			accountID, ecrRegionName, ecrRepositoryName, ecrImageTag,
		)

		pterm.Info.Println("Pushing image to AWS ECR...")

		if err := docker.PushImageToECR(ecrImage, ecrRegionName, ecrRepositoryName); err != nil {
			pterm.Error.Println("Failed to push image to ECR:", err)
			return err
		}
		pterm.Success.Println("Successfully pushed image to ECR:", ecrImage)

		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", imageRef)
			if err := docker.RemoveImage(imageRef); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", imageRef)
		}

		return nil
	},
	Example: `
  # IMAGE_NAME can be in the form:
  #   123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo:python

  smurf sdkr push aws 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-repo:python
  smurf sdkr push aws 123456789012.dkr.ecr.us-east-1.amazonaws.com/repo-name:python --delete
`,
}

func init() {
	pushEcrCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false,
		"Delete the local image after pushing",
	)

	pushCmd.AddCommand(pushEcrCmd)
}
