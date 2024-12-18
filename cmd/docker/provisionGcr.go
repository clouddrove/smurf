package docker

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// Flags for the provisionGcr command
var (
	provisionGcrProjectID       string
	provisionGcrImageName       string
	provisionGcrImageTag        string
	provisionGcrDockerfilePath  string
	provisionGcrNoCache         bool
	provisionGcrBuildArgs       []string
	provisionGcrTarget          string
	provisionGcrSarifFile       string
	provisionGcrTargetTag       string
	provisionGcrConfirmPush     bool
	provisionGcrDeleteAfterPush bool
	provisionGcrPlatform        string
	provisionGcrAuto            bool
)

var provisionGcrCmd = &cobra.Command{
	Use:   "provision-gcr",
	Short: "Build, scan, tag, and push a Docker image to Google Container Registry.",
	Long: `Build, scan, tag, and push a Docker image to Google Container Registry.
	Set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of your service account JSON key file.
	export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if provisionGcrAuto {

			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			var envVars map[string]string

			if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
				envVars = map[string]string{
					"GOOGLE_APPLICATION_CREDENTIALS": data.Sdkr.GoogleApplicationCredentials,
				}
			}

			if err := configs.ExportEnvironmentVariables(envVars); err != nil {
				fmt.Println("Error exporting variables:", err)
				return err
			}

			sampleImageNameForGcr := "my-image"

			if provisionGcrImageName == "" {
				provisionGcrImageName = sampleImageNameForGcr
			}

			if provisionGcrProjectID == "" {
				provisionGcrProjectID = data.Sdkr.ProvisionGcrProjectID
			}
		}

		if provisionGcrProjectID == "" || provisionGcrImageName == "" {
			pterm.Error.Println("Required flags are missing. Please provide the required flags.")
		}

		fullGcrImage := fmt.Sprintf("gcr.io/%s/%s:%s", provisionGcrProjectID, provisionGcrImageName, provisionGcrImageTag)

		buildArgsMap := make(map[string]string)
		for _, arg := range provisionGcrBuildArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		buildOpts := docker.BuildOptions{
			DockerfilePath: provisionGcrDockerfilePath,
			NoCache:        provisionGcrNoCache,
			BuildArgs:      buildArgsMap,
			Target:         provisionGcrTarget,
			Platform:       provisionGcrPlatform,
		}

		pterm.Info.Println("Starting GCR build...")
		if err := docker.Build(provisionGcrImageName, provisionGcrImageTag, buildOpts); err != nil {
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
			scanErr = docker.Scout(fullGcrImage, provisionGcrSarifFile)
			if scanErr != nil {
				pterm.Error.Println("Scan failed:", scanErr)
			} else {
				pterm.Success.Println("Scan completed successfully.")
			}
		}()

		go func() {
			defer wg.Done()
			if provisionGcrTargetTag != "" {
				pterm.Info.Printf("Tagging image as %s...\n", provisionGcrTargetTag)
				tagOpts := docker.TagOptions{
					Source: fullGcrImage,
					Target: provisionGcrTargetTag,
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
			return fmt.Errorf("GCR provisioning failed due to previous errors")
		}

		pushImage := provisionGcrTargetTag
		if pushImage == "" {
			pushImage = fullGcrImage
		}

		if provisionGcrConfirmPush {
			pterm.Info.Printf("Pushing image %s to GCR...\n", pushImage)
			if err := docker.PushImageToGCR(provisionGcrProjectID, provisionGcrImageName); err != nil {
				pterm.Error.Println("Push to GCR failed:", err)
				return err
			}
			pterm.Success.Println("Push to GCR completed successfully.")
		}

		if provisionGcrDeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullGcrImage)
			if err := docker.RemoveImage(fullGcrImage); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", fullGcrImage)
		}

		pterm.Success.Println("GCR provisioning completed successfully.")
		return nil
	},
	Example: `
	smurf sdkr provision-gcr --project-id my-project --image-name my-image --tag my-tag
	smurf sdkr provision-gcr --project-id my-project --image-name my-image --tag my-tag --file Dockerfile --no-cache --build-arg key1=value1 --build-arg key2=value2 --target my-target --output my-sarif-file.sarif --target-tag my-target-tag --yes --delete --platform linux/amd64`,
}

func init() {
	provisionGcrCmd.Flags().StringVarP(&provisionGcrProjectID, "project-id", "p", "", "GCP project ID (required)")
	provisionGcrCmd.Flags().StringVarP(&provisionGcrImageName, "image-name", "i", "", "Name of the image to build")
	provisionGcrCmd.Flags().StringVarP(&provisionGcrImageTag, "tag", "t", "latest", "Tag for the image")
	provisionGcrCmd.Flags().StringVarP(&provisionGcrDockerfilePath, "file", "f", "Dockerfile", "Name of the Dockerfile (default is 'Dockerfile')")
	provisionGcrCmd.Flags().BoolVar(&provisionGcrNoCache, "no-cache", false, "Do not use cache when building the image")
	provisionGcrCmd.Flags().StringArrayVar(&provisionGcrBuildArgs, "build-arg", []string{}, "Set build-time variables")
	provisionGcrCmd.Flags().StringVar(&provisionGcrTarget, "target", "", "Set the target build stage to build")
	provisionGcrCmd.Flags().StringVarP(&provisionGcrSarifFile, "output", "o", "", "Output file for SARIF report")
	provisionGcrCmd.Flags().StringVar(&provisionGcrTargetTag, "target-tag", "", "Target tag for tagging the image")
	provisionGcrCmd.Flags().BoolVarP(&provisionGcrConfirmPush, "yes", "y", false, "Push the image to GCR without confirmation")
	provisionGcrCmd.Flags().BoolVarP(&provisionGcrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	provisionGcrCmd.Flags().StringVar(&provisionGcrPlatform, "platform", "", "Set the platform for the image")
	provisionGcrCmd.Flags().BoolVar(&provisionGcrAuto, "auto", false, "Automatically push the image to GCR after tagging")

	sdkrCmd.AddCommand(provisionGcrCmd)
}
