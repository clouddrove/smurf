package sdkr

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// provisionAcrCmd sets up the "provision-acr" command, enabling the build,
// and optional push of a Docker image to Azure Container Registry.
// It supports default values from the config file if no arguments are provided,
// as well as advanced features like specifying build args, timeouts, and
// removing the local image once it's successfully pushed.
var provisionAcrCmd = &cobra.Command{
	Use:          "provision-acr [IMAGE_NAME[:TAG]]",
	Short:        "Build and push a Docker image to Azure Container Registry.",
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

		if configs.SubscriptionID == "" || configs.ResourceGroup == "" || configs.RegistryName == "" {
			pterm.Error.Println("Azure subscription ID, resource group name, and registry name are required")
			return errors.New("azure subscription ID, resource group name, and registry name are required")
		}

		fullAcrImage := fmt.Sprintf("%s.azurecr.io/%s", configs.RegistryName, imageRef)

		buildArgsMap := make(map[string]string)
		for _, arg := range configs.BuildArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		if configs.ContextDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				pterm.Error.Printfln("failed to get current working directory: %v", err)
				return fmt.Errorf("failed to get current working directory: %w", err)
			}
			configs.ContextDir = wd
		}

		if configs.DockerfilePath == "" {
			configs.DockerfilePath = filepath.Join(configs.ContextDir, "Dockerfile")
		} else {
			configs.DockerfilePath = filepath.Join(configs.ContextDir, configs.DockerfilePath)
		}

		buildOpts := docker.BuildOptions{
			ContextDir:     configs.ContextDir,
			DockerfilePath: configs.DockerfilePath,
			NoCache:        configs.NoCache,
			BuildArgs:      buildArgsMap,
			Target:         configs.Target,
			Platform:       configs.Platform,
			Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
		}

		pterm.Info.Println("Starting ACR build...")
		localImageName, localTag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			pterm.Error.Println("Image Parse Err:", parseErr)
			return parseErr
		}

		if localTag == "" {
			localTag = "latest"
		}

		if err := docker.Build(localImageName, localTag, buildOpts, useAI); err != nil {
			return err
		}
		pterm.Success.Println("Build completed successfully.")

		pushImage := imageRef
		if pushImage == "" {
			pushImage = fullAcrImage
		}

		if !configs.ConfirmAfterPush {
			pterm.Info.Println("Press Enter to continue...")
			buf := bufio.NewReader(os.Stdin)
			_, _ = buf.ReadBytes('\n')
		}

		pterm.Info.Printf("Pushing image %s to ACR...\n", pushImage)
		if err := docker.PushImageToACR(
			configs.SubscriptionID,
			configs.ResourceGroup,
			configs.RegistryName,
			localImageName,
			useAI,
		); err != nil {
			pterm.Error.Println("Push to ACR failed:", err)
			return err
		}
		pterm.Success.Println("Push to ACR completed successfully.")

		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullAcrImage)
			if err := docker.RemoveImage(fullAcrImage, useAI); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return fmt.Errorf("failed to delete local image: %v", err)
			}
			pterm.Success.Println("Successfully deleted local image:", fullAcrImage)
		}

		pterm.Success.Println("ACR provisioning completed successfully.")
		return nil
	},
	Example: `
  smurf sdkr provision-acr myimage:v1 -s <SUBSCRIPTION_ID> -r <RESOURCE_GROUP> -g <REGISTRY_NAME>
  smurf sdkr provision-acr -f Dockerfile -c -a key1=value1 -a key2=value2 -t my-target -p linux/amd64 -o report.sarif  -y -d
`,
}

func init() {
	provisionAcrCmd.Flags().StringVarP(&configs.SubscriptionID, "subscription-id", "s", "", "Azure subscription ID (required)")
	provisionAcrCmd.Flags().StringVarP(&configs.ResourceGroup, "resource-group", "r", "", "Azure resource group name (required)")
	provisionAcrCmd.Flags().StringVarP(&configs.RegistryName, "registry-name", "g", "", "Azure Container Registry name (required)")

	provisionAcrCmd.Flags().StringVarP(&configs.DockerfilePath, "file", "f", "", "path to Dockerfile relative to context directory")
	provisionAcrCmd.Flags().BoolVarP(&configs.NoCache, "no-cache", "c", false, "Do not use cache when building the image")
	provisionAcrCmd.Flags().StringArrayVarP(&configs.BuildArgs, "build-arg", "a", []string{}, "Set build-time variables")
	provisionAcrCmd.Flags().StringVarP(&configs.Target, "target", "t", "", "Set the target build stage to build")
	provisionAcrCmd.Flags().StringVarP(&configs.Platform, "platform", "p", "", "Platform for the image")
	provisionAcrCmd.Flags().StringVar(&configs.ContextDir, "context", "", "Build context directory (default: current directory)")
	provisionAcrCmd.Flags().IntVar(&configs.BuildTimeout, "timeout", 1500, "Build timeout")

	provisionAcrCmd.Flags().StringVarP(&configs.SarifFile, "output", "o", "", "Output file for SARIF report")

	provisionAcrCmd.Flags().BoolVarP(&configs.ConfirmAfterPush, "yes", "y", false, "Push the image to ACR without confirmation")
	provisionAcrCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	provisionAcrCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	sdkrCmd.AddCommand(provisionAcrCmd)
}
