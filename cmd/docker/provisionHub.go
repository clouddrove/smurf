package docker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// Flags for the provision command
var (
	provisionImageName      string
	provisionImageTag       string
	provisionDockerfilePath string
	provisionNoCache        bool
	provisionBuildArgs      []string
	provisionTarget         string
	provisionSarifFile      string
	provisionTargetTag      string
	provisionConfirmPush    bool
	provisionDeleteAfterPush bool
	provisionPlatform       string
)

var provisionHubCmd = &cobra.Command{
	Use:   "provision-hub",
	Short: "Build, scan, tag, and push a Docker image.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fullImageName := fmt.Sprintf("%s:%s", provisionImageName, provisionImageTag)

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

	provisionHubCmd.MarkFlagRequired("image-name")

	sdkrCmd.AddCommand(provisionHubCmd)
}
