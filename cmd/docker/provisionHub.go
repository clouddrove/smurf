package docker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// Flags for the provision command
var (
	provisionImageName       string
	provisionImageTag        string
	provisionDockerfilePath  string
	provisionNoCache         bool
	provisionBuildArgs       []string
	provisionTarget          string
	provisionSarifFile       string
	provisionTargetTag       string
	provisionConfirmPush     bool
	provisionDeleteAfterPush bool
	provisionPlatform        string
	provisionAuto            bool
)

var provisionHubCmd = &cobra.Command{
	Use:   "provision-hub",
	Short: "Build, scan, tag, and push a Docker image.",
	Long: `Build, scan, tag, and push a Docker image.
	export DOCKER_USERNAME="your-username" 
	export DOCKER_PASSWORD="your-password"
	for Docker Hub authentication.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fullImageName := fmt.Sprintf("%s:%s", provisionImageName, provisionImageTag)

		if provisionAuto {
			args = append(args, "my-image", "latest")

			data, err := configs.LoadConfig(configs.FileName)

			if err != nil {
				return err
			}

			envVars := map[string]string{
				"DOCKER_USERNAME": data.DockerUsername,
				"DOCKER_PASSWORD": data.DockerPassword,
			}

			if err := configs.ExportEnvironmentVariables(envVars); err != nil {
				fmt.Println("Error exporting variables:", err)
				return err
			}

		}

		buildArgsMap := make(map[string]string)
		for _, arg := range provisionBuildArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		buildOpts := docker.BuildOptions{
			DockerfilePath: provisionDockerfilePath,
			NoCache:        provisionNoCache,
			BuildArgs:      buildArgsMap,
			Target:         provisionTarget,
			Platform:       provisionPlatform,
		}

		pterm.Info.Println("Starting build...")
		if err := docker.Build(provisionImageName, provisionImageTag, buildOpts); err != nil {
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
			scanErr = docker.Scout(fullImageName, provisionSarifFile)
			if scanErr != nil {
				pterm.Error.Println("Scan failed:", scanErr)
			} else {
				pterm.Success.Println("Scan completed successfully.")
			}
		}()

		go func() {
			defer wg.Done()
			if provisionTargetTag != "" {
				pterm.Info.Printf("Tagging image as %s...\n", provisionTargetTag)
				tagOpts := docker.TagOptions{
					Source: fullImageName,
					Target: provisionTargetTag,
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
			return fmt.Errorf("provisioning failed due to previous errors")
		}

		pushImage := provisionTargetTag
		if pushImage == "" {
			pushImage = fullImageName
		}

		if provisionConfirmPush {
			pterm.Info.Printf("Pushing image %s...\n", pushImage)
			pushOpts := docker.PushOptions{
				ImageName: pushImage,
			}
			if err := docker.PushImage(pushOpts); err != nil {
				pterm.Error.Println("Push failed:", err)
				return err
			}
			pterm.Success.Println("Push completed successfully.")
		} else {
			result, _ := pterm.DefaultInteractiveConfirm.
				WithDefaultText("Do you want to push the image?").
				Show()
			if result {
				pterm.Info.Printf("Pushing image %s...\n", pushImage)
				pushOpts := docker.PushOptions{
					ImageName: pushImage,
				}
				if err := docker.PushImage(pushOpts); err != nil {
					pterm.Error.Println("Push failed:", err)
					return err
				}
				pterm.Success.Println("Push completed successfully.")
			} else {
				pterm.Info.Println("Image push skipped.")
			}
		}

		if provisionDeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullImageName)
			if err := docker.RemoveImage(fullImageName); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", fullImageName)
		}

		pterm.Success.Println("Provisioning completed successfully.")
		return nil
	},
	Example: `
	smurf sdkr provision-hub --image-name my-image --tag my-tag
	smurf sdkr provision-hub --image-name my-image --tag my-tag --file Dockerfile --no-cache --build-arg key1=value1 --build-arg key2=value2 --target my-target --platform linux/amd64 --output report.sarif --target-tag my-tag --yes --delete
	`,
}

func init() {
	provisionHubCmd.Flags().StringVarP(&provisionImageName, "image-name", "i", "", "Name of the image to build")
	provisionHubCmd.Flags().StringVarP(&provisionImageTag, "tag", "t", "latest", "Tag for the image")
	provisionHubCmd.Flags().StringVarP(&provisionDockerfilePath, "file", "f", "Dockerfile", "Name of the Dockerfile (default is 'Dockerfile')")
	provisionHubCmd.Flags().BoolVar(&provisionNoCache, "no-cache", false, "Do not use cache when building the image")
	provisionHubCmd.Flags().StringArrayVar(&provisionBuildArgs, "build-arg", []string{}, "Set build-time variables")
	provisionHubCmd.Flags().StringVar(&provisionTarget, "target", "", "Set the target build stage to build")
	provisionHubCmd.Flags().StringVarP(&provisionSarifFile, "output", "o", "", "Output file for SARIF report")
	provisionHubCmd.Flags().StringVar(&provisionTargetTag, "target-tag", "", "Target tag for tagging the image")
	provisionHubCmd.Flags().BoolVarP(&provisionConfirmPush, "yes", "y", false, "Push the image without confirmation")
	provisionHubCmd.Flags().BoolVarP(&provisionDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	provisionHubCmd.Flags().StringVar(&provisionPlatform, "platform", "", "Set the platform for the image")
	provisionHubCmd.Flags().BoolVarP(&provisionAuto, "auto", "a", false, "Auto provision the image")

	provisionHubCmd.MarkFlagRequired("image-name")

	sdkrCmd.AddCommand(provisionHubCmd)
}
