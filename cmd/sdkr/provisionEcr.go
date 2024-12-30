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

// provisionEcrCmd orchestrates the steps to build a Docker image locally,
// optionally scan it for vulnerabilities, and then push it to AWS ECR.
// It relies on config defaults or command-line flags for region, repository,
// and other Docker build settings.
var provisionEcrCmd = &cobra.Command{
	Use:   "provision-ecr [IMAGE_NAME[:TAG]]",
	Short: "Build, scan, and push a Docker image to AWS ECR.",
	Args:  cobra.MaximumNArgs(1),
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
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName

			if configs.Region == "" {
				configs.Region = data.Sdkr.ProvisionEcrRegion
			}
			if configs.Repository == "" {
				configs.Repository = data.Sdkr.ProvisionEcrRepository
			}
		}

		if configs.Region == "" || configs.RegistryName == "" {
			pterm.Error.Println("AWS region and ECR repository name are required")
			return errors.New("missing required AWS ECR parameters")
		}

		localImageName, localTag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			return fmt.Errorf("invalid image format: %w", parseErr)
		}

		if localTag == "" {
			localTag = "latest"
		}

		fullEcrImage := fmt.Sprintf(
			"%s.dkr.ecr.%s.amazonaws.com/%s:%s",
			localImageName, 
			configs.Region,
			configs.Repository,
			localTag,
		)

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

		pterm.Info.Println("Starting ECR build...")
		if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
			pterm.Error.Println("Build failed:", err)
			return err
		}
		pterm.Success.Println("Build completed successfully.")

		scanErr := docker.Scout(fullEcrImage, configs.SarifFile)
		if scanErr != nil {
			pterm.Error.Println("Scan failed:", scanErr)
		} else {
			pterm.Success.Println("Scan completed successfully.")
		}

		if scanErr != nil {
			return fmt.Errorf("ECR provisioning failed due to previous errors")
		}

		pushImage := imageRef
		if pushImage == "" {
			pushImage = fullEcrImage
		}

		if !configs.ConfirmAfterPush {
			pterm.Info.Println("Press Enter to continue...")
			buf := bufio.NewReader(os.Stdin)
			_, _ = buf.ReadBytes('\n')
		}

		if configs.ConfirmAfterPush {
			pterm.Info.Printf("Pushing image %s to ECR...\n", pushImage)
			if err := docker.PushImageToECR(localImageName, configs.Region, configs.Repository); err != nil {
				pterm.Error.Println("Push to ECR failed:", err)
				return err
			}
			pterm.Success.Println("Push to ECR completed successfully.")
		}

		

		if configs.DeleteAfterPush {
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
	Example: `
  # Example: Provide "accountID/myimage:tag" as an argument, region & repository as flags
  smurf sdkr provision-ecr my-image:v1 \
      -r us-east-1 \
      -R my-ecr-repository \
      --no-cache \
      --build-arg key1=value1 \
      --build-arg key2=value2 \
      --target my-target \
      --platform linux/amd64 \
      --output scan.sarif \
      --yes \
      --delete
`,
}

func init() {
	provisionEcrCmd.Flags().StringVarP(&configs.DockerfilePath, "file", "f", "", "Dockerfile path relative to context directory (default: 'Dockerfile')")
	provisionEcrCmd.Flags().BoolVarP(&configs.NoCache, "no-cache", "c", false, "Do not use cache when building the image")
	provisionEcrCmd.Flags().StringArrayVarP(&configs.BuildArgs, "build-arg", "a", []string{}, "Set build-time variables")
	provisionEcrCmd.Flags().StringVarP(&configs.Target, "target", "T", "", "Set the target build stage to build")
	provisionEcrCmd.Flags().StringVarP(&configs.Platform, "platform", "p", "", "Platform for the image")

	provisionEcrCmd.Flags().StringVar(&configs.ContextDir, "context", "", "Build context directory (default: current directory)")
	provisionEcrCmd.Flags().IntVar(&configs.BuildTimeout, "timeout", 1500, "Build timeout")

	provisionEcrCmd.Flags().StringVarP(&configs.SarifFile, "output", "o", "", "Output file for SARIF report")

	provisionEcrCmd.Flags().BoolVarP(&configs.ConfirmAfterPush, "yes", "y", false, "Push the image to ECR without confirmation")
	provisionEcrCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")

	provisionEcrCmd.Flags().StringVarP(&configs.Region, "region", "r", "", "AWS region (required)")
	provisionEcrCmd.Flags().StringVarP(&configs.Repository, "repository", "R", "", "AWS ECR repository name (required)")

	sdkrCmd.AddCommand(provisionEcrCmd)
}
