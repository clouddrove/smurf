package sdkr

import (
	"errors"
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// pushAcrCmd defines a subcommand that pushes a Docker image to Azure Container Registry (ACR).
// It supports both direct arguments and config file-based parameters for the image reference,
// as well as optional removal of the local image after a successful push.
var pushAcrCmd = &cobra.Command{
	Use:          "az [IMAGE_NAME[:TAG]]",
	Short:        "Push a Docker image to Azure Container Registry.",
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
				return errors.New(pterm.Error.Sprintfln("image name (with optional tag) must be provided either as an argument or in the config"))
			}
			imageRef = data.Sdkr.ImageName

			if configs.SubscriptionID == "" {
				configs.SubscriptionID = data.Sdkr.ProvisionAcrSubscriptionID
			}
			if configs.ResourceGroup == "" {
				configs.ResourceGroup = data.Sdkr.ProvisionAcrResourceGroup
			}
			if configs.RegistryName == "" {
				configs.RegistryName = data.Sdkr.ProvisionAcrRegistryName
			}
		}

		repoName, tag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			pterm.Error.Printfln("invalid image format: %v", parseErr)
			return fmt.Errorf("invalid image format: %v", parseErr)
		}
		if repoName == "" {
			return errors.New("invalid image reference")
		}
		if tag == "" {
			tag = "latest"
		}

		if configs.SubscriptionID == "" || configs.ResourceGroup == "" || configs.RegistryName == "" || repoName == "" {
			pterm.Error.Println("Required flags are missing. Please provide the required flags.")
			return errors.New("missing required ACR parameters")
		}

		acrImage := fmt.Sprintf("%s.azurecr.io/%s:%s", configs.RegistryName, repoName, tag)

		pterm.Info.Println("Pushing image to Azure Container Registry...")
		if err := docker.PushImageToACR(configs.SubscriptionID, configs.ResourceGroup, configs.RegistryName, repoName); err != nil {
			pterm.Error.Println("Failed to push image:", err)
			return err
		}
		pterm.Success.Println("Successfully pushed image to ACR:", acrImage)

		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", repoName)
			if err := docker.RemoveImage(repoName); err != nil {
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", repoName)
		}

		return nil
	},
	Example: `
  smurf sdkr push az myapp:v1 -s <subscription-id> -r <resource-group> -g <registry-name> --delete
  smurf sdkr push az myapp:v1 --subscription-id <subscription-id> --resource-group <resource-group> --registry-name <registry-name>
  `,
}

func init() {
	pushAcrCmd.Flags().StringVarP(&configs.SubscriptionID, "subscription-id", "s", "", "Azure subscription ID (required)")
	pushAcrCmd.Flags().StringVarP(&configs.ResourceGroup, "resource-group", "r", "", "Azure resource group name (required)")
	pushAcrCmd.Flags().StringVarP(&configs.RegistryName, "registry-name", "g", "", "Azure Container Registry name (required)")
	pushAcrCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	pushCmd.AddCommand(pushAcrCmd)
}
