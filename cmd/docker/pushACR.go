package docker

import (
	"fmt"

	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	acrSubscriptionID  string
	acrResourceGroup   string
	acrRegistryName    string
	acrImageName       string
	acrImageTag        string
	acrDeleteAfterPush bool
)

var pushAcrCmd = &cobra.Command{
	Use:   "az",
	Short: "push docker images to acr",
	RunE: func(cmd *cobra.Command, args []string) error {
		if acrSubscriptionID == "" || acrResourceGroup == "" || acrRegistryName == "" {
			return fmt.Errorf("azure requires --subscription-id, --resource-group, and --registry-name flags")
		}

		acrImage := fmt.Sprintf("%s.azurecr.io/%s:%s", acrRegistryName, acrImageName, acrImageTag)

		pterm.Info.Println("Pushing image to Azure Container Registry...")
		if err := docker.PushImageToACR(acrSubscriptionID, acrResourceGroup, acrRegistryName, acrImageName); err != nil {
			return err
		}
		pterm.Success.Println("Successfully pushed image to ACR:", acrImage)

		if acrDeleteAfterPush {
			if err := docker.RemoveImage(acrImageName); err != nil {
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", acrImageName)
		}

		return nil
	},
}

func init() {
	pushAcrCmd.Flags().StringVarP(&acrImageName, "image", "i", "", "Image name (e.g., myapp)")
	pushAcrCmd.Flags().StringVarP(&acrImageTag, "tag", "t", "latest", "Image tag (default: latest)")
	pushAcrCmd.Flags().BoolVarP(&acrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	pushAcrCmd.Flags().StringVar(&acrSubscriptionID, "subscription-id", "", "Azure subscription ID (required with --azure)")
	pushAcrCmd.Flags().StringVar(&acrResourceGroup, "resource-group", "", "Azure resource group name (required with --azure)")
	pushAcrCmd.Flags().StringVar(&acrRegistryName, "registry-name", "", "Azure Container Registry name (required with --azure)")

	pushAcrCmd.MarkFlagRequired("subscription-id")
	pushAcrCmd.MarkFlagRequired("resource-group")
	pushAcrCmd.MarkFlagRequired("registry-name")
	pushAcrCmd.MarkFlagRequired("image")

	pushCmd.AddCommand(pushAcrCmd)
}
