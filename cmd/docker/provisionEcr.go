package docker

import (
	"fmt"
	"strings"
	"sync"

	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// Flags for the provisionEcr command
var (
	provisionEcrImageName      string
	provisionEcrImageTag       string
	provisionEcrDockerfilePath string
	provisionEcrNoCache        bool
	provisionEcrBuildArgs      []string
	provisionEcrTarget         string
	provisionEcrSarifFile      string
	provisionEcrTargetTag      string
	provisionEcrConfirmPush    bool
	provisionEcrDeleteAfterPush bool
	provisionEcrRegion         string
	provisionEcrRepository     string
	provisionEcrPlatform       string
)

var provisionEcrCmd = &cobra.Command{
	Use:   "provision-ecr",
	Short: "Build, scan, tag, and push a Docker image to AWS ECR.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if provisionEcrRegion == "" || provisionEcrRepository == "" {
			return fmt.Errorf("ECR provisioning requires both --region and --repository flags")
		}

		fullEcrImage := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s", provisionEcrImageName, provisionEcrRegion, provisionEcrRepository, provisionEcrImageTag)

		buildArgsMap := make(map[string]string)
		for _, arg := range provisionEcrBuildArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		buildOpts := docker.BuildOptions{
			DockerfilePath: provisionEcrDockerfilePath,
			NoCache:        provisionEcrNoCache,
			BuildArgs:      buildArgsMap,
			Target:         provisionEcrTarget,
			Platform:       provisionEcrPlatform,
		}

		pterm.Info.Println("Starting ECR build...")
		if err := docker.Build(provisionEcrImageName, provisionEcrImageTag, buildOpts); err != nil {
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
			scanErr = docker.Scout(fullEcrImage, provisionEcrSarifFile)
			if scanErr != nil {
				pterm.Error.Println("Scan failed:", scanErr)
			} else {
				pterm.Success.Println("Scan completed successfully.")
			}
		}()

		go func() {
			defer wg.Done()
			if provisionEcrTargetTag != "" {
				pterm.Info.Printf("Tagging image as %s...\n", provisionEcrTargetTag)
				tagOpts := docker.TagOptions{
					Source: fullEcrImage,
					Target: provisionEcrTargetTag,
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
			return fmt.Errorf("ECR provisioning failed due to previous errors")
		}

		pushImage := provisionEcrTargetTag
		if pushImage == "" {
			pushImage = fullEcrImage
		}

		if provisionEcrConfirmPush {
			pterm.Info.Printf("Pushing image %s to ECR...\n", pushImage)
			if err := docker.PushImageToECR(provisionEcrImageName, provisionEcrRegion, provisionEcrRepository); err != nil {
				pterm.Error.Println("Push to ECR failed:", err)
				return err
			}
			pterm.Success.Println("Push to ECR completed successfully.")
		}

		if provisionEcrDeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullEcrImage)
			if err := docker.RemoveImage(fullEcrImage); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", fullEcrImage)
		}

		pterm.Success.Println("ECR provisioning completed successfully.")
		return nil
	},
}

func init() {
	provisionEcrCmd.Flags().StringVarP(&provisionEcrImageName, "image-name", "i", "", "Name of the image to build")
	provisionEcrCmd.Flags().StringVarP(&provisionEcrImageTag, "tag", "t", "latest", "Tag for the image")
	provisionEcrCmd.Flags().StringVarP(&provisionEcrDockerfilePath, "file", "f", "Dockerfile", "Name of the Dockerfile (default is 'Dockerfile')")
	provisionEcrCmd.Flags().BoolVar(&provisionEcrNoCache, "no-cache", false, "Do not use cache when building the image")
	provisionEcrCmd.Flags().StringArrayVar(&provisionEcrBuildArgs, "build-arg", []string{}, "Set build-time variables")
	provisionEcrCmd.Flags().StringVar(&provisionEcrTarget, "target", "", "Set the target build stage to build")
	provisionEcrCmd.Flags().StringVarP(&provisionEcrSarifFile, "output", "o", "", "Output file for SARIF report")
	provisionEcrCmd.Flags().StringVar(&provisionEcrTargetTag, "target-tag", "", "Target tag for tagging the image")
	provisionEcrCmd.Flags().BoolVarP(&provisionEcrConfirmPush, "yes", "y", false, "Push the image to ECR without confirmation")
	provisionEcrCmd.Flags().BoolVarP(&provisionEcrDeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	provisionEcrCmd.Flags().StringVarP(&provisionEcrRegion, "region", "r", "", "AWS region (required)")
	provisionEcrCmd.Flags().StringVarP(&provisionEcrRepository, "repository", "R", "", "AWS ECR repository name (required)")
	provisionEcrCmd.Flags().StringVar(&provisionEcrPlatform, "platform", "", "Platform for the build")

	provisionEcrCmd.MarkFlagRequired("image-name")
	provisionEcrCmd.MarkFlagRequired("region")
	provisionEcrCmd.MarkFlagRequired("repository")

	sdkrCmd.AddCommand(provisionEcrCmd)
}
