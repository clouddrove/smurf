package docker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// Flags for the provisionAcr command
var (
	provisionAcrSubscriptionID  string
	provisionAcrResourceGroup   string
	provisionAcrRegistryName    string
	provisionAcrImageName       string
	provisionAcrImageTag        string
	provisionAcrDockerfilePath  string
	provisionAcrNoCache         bool
	provisionAcrBuildArgs       []string
	provisionAcrTarget          string
	provisionAcrSarifFile       string
	provisionAcrTargetTag       string
	provisionAcrConfirmPush     bool
	provisionAcrDeleteAfterPush bool
	provisionAcrPlatform        string
)

var provisionAcrCmd = &cobra.Command{
	Use:   "provision-acr",
	Short: "Build, scan, tag, and push a Docker image to Azure Container Registry.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if provisionAcrSubscriptionID == "" || provisionAcrResourceGroup == "" || provisionAcrRegistryName == "" {
			return fmt.Errorf("ACR provisioning requires --subscription-id, --resource-group, and --registry-name flags")
		}

		fullAcrImage := fmt.Sprintf("%s.azurecr.io/%s:%s", provisionAcrRegistryName, provisionAcrImageName, provisionAcrImageTag)

		buildArgsMap := make(map[string]string)
		for _, arg := range provisionAcrBuildArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		buildOpts := docker.BuildOptions{
			DockerfilePath: provisionAcrDockerfilePath,
			NoCache:        provisionAcrNoCache,
			BuildArgs:      buildArgsMap,
			Target:         provisionAcrTarget,
			Platform:	   provisionAcrPlatform,
		}

		pterm.Info.Println("Starting ACR build...")
		if err := docker.Build(provisionAcrImageName, provisionAcrImageTag, buildOpts); err != nil {
			pterm.Error.Println("Build failed:", err)
			return err
		}
		pterm.Success.Println("Build completed successfully.")

		var wg sync.WaitGroup
		var scanErr, tagErr error

		wg.Add(2)

		go func() {
			defer wg.Done()
			pterm.Info.Println("Starting scan...")
			scanErr = docker.Scout(fullAcrImage, provisionAcrSarifFile)
			if scanErr != nil {
				pterm.Error.Println("Scan failed:", scanErr)
			} else {
				pterm.Success.Println("Scan completed successfully.")
			}
		}()

		go func() {
			defer wg.Done()
			if provisionAcrTargetTag != "" {
				pterm.Info.Printf("Tagging image as %s...\n", provisionAcrTargetTag)
				tagOpts := docker.TagOptions{
					Source: fullAcrImage,
					Target: provisionAcrTargetTag,
				}
				tagErr = docker.TagImage(tagOpts)
				if tagErr != nil {
					pterm.Error.Println("Tagging failed:", tagErr)
				} else {
					pterm.Success.Println("Tagging completed successfully.")
				}
			}
		}()

		wg.Wait()

		if scanErr != nil || tagErr != nil {
			return fmt.Errorf("ACR provisioning failed due to previous errors")
		}

		pushImage := provisionAcrTargetTag
		if pushImage == "" {
			pushImage = fullAcrImage
		}

		if provisionAcrConfirmPush {
			pterm.Info.Printf("Pushing image %s to ACR...\n", pushImage)
			if err := docker.PushImageToACR(provisionAcrSubscriptionID, provisionAcrResourceGroup, provisionAcrRegistryName, provisionAcrImageName); err != nil {
				pterm.Error.Println("Push to ACR failed:", err)
				return err
			}
			pterm.Success.Println("Push to ACR completed successfully.")
		}

		if provisionAcrDeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullAcrImage)
			if err := docker.RemoveImage(fullAcrImage); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", fullAcrImage)
		}

		pterm.Success.Println("ACR provisioning completed successfully.")
		return nil
	},
}

func init() {
	provisionAcrCmd.Flags().StringVarP(&provisionAcrImageName, "image-name", "i", "", "Name of the image to build")
	provisionAcrCmd.Flags().StringVarP(&provisionAcrImageTag, "tag", "t", "latest", "Tag for the image")
	provisionAcrCmd.Flags().StringVarP(&provisionAcrDockerfilePath, "file", "f", "Dockerfile", "Name of the Dockerfile (default is 'Dockerfile')")
	provisionAcrCmd.Flags().BoolVar(&provisionAcrNoCache, "no-cache", false, "Do not use cache when building the image")
	provisionAcrCmd.Flags().StringArrayVar(&provisionAcrBuildArgs, "build-arg", []string{}, "Set build-time variables")
	provisionAcrCmd.Flags().StringVar(&provisionAcrTarget, "target", "", "Set the target build stage to build")
	provisionAcrCmd.Flags().StringVarP(&provisionAcrSarifFile, "output", "o", "", "Output file for SARIF report")
	provisionAcrCmd.Flags().StringVar(&provisionAcrTargetTag, "target-tag", "", "Target tag for tagging the image")
	provisionAcrCmd.Flags().BoolVarP(&provisionAcrConfirmPush, "yes", "y", false, "Push the image to ACR without confirmation")
	provisionAcrCmd.Flags().BoolVarP(&provisionAcrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	provisionAcrCmd.Flags().StringVar(&provisionAcrSubscriptionID, "subscription-id", "", "Azure subscription ID (required)")
	provisionAcrCmd.Flags().StringVar(&provisionAcrResourceGroup, "resource-group", "", "Azure resource group name (required)")
	provisionAcrCmd.Flags().StringVar(&provisionAcrRegistryName, "registry-name", "", "Azure Container Registry name (required)")
	provisionAcrCmd.Flags().StringVar(&provisionAcrPlatform, "platform", "", "Platform for the image")

	provisionAcrCmd.MarkFlagRequired("subscription-id")
	provisionAcrCmd.MarkFlagRequired("resource-group")
	provisionAcrCmd.MarkFlagRequired("registry-name")
	provisionAcrCmd.MarkFlagRequired("image-name")

	sdkrCmd.AddCommand(provisionAcrCmd)
}
