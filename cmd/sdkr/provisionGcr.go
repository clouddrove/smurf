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


// provisionGcrCmd encapsulates the logic to build a Docker image, run a security scan, 
// and optionally push that image to Google Container Registry. It also leverages 
// environment variables or config file values for project credentials and other 
// parameters. The command supports features like custom build args, timeouts, 
// and conditional deletion of local images post-push.
var provisionGcrCmd = &cobra.Command{
	Use:   "provision-gcr [IMAGE_NAME[:TAG]]",
	Short: "Build, scan, tag, and push a Docker image to Google Container Registry.",
	Long: `Build, scan, tag, and push a Docker image to Google Container Registry.
Set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of your service account JSON key file, for example:
  export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/service-account-key.json"
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageRef string
		var envVars map[string]string

		if len(args) == 1 {
			imageRef = args[0] 
		} else {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
				envVars = map[string]string{
					"GOOGLE_APPLICATION_CREDENTIALS": data.Sdkr.GoogleApplicationCredentials,
				}
			}
			if envVars != nil {
				if err := configs.ExportEnvironmentVariables(envVars); err != nil {
					return err
				}
			}

			if envVars["GOOGLE_APPLICATION_CREDENTIALS"] == "" {
				pterm.Error.Println("Google Application Credentials is required")
				return errors.New("missing required Google Application Credentials")
			}

			if data.Sdkr.ImageName == "" {
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName
		}

		if configs.ProjectID == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if data.Sdkr.ProvisionGcrProjectID != "" {
				configs.ProjectID = data.Sdkr.ProvisionGcrProjectID
			}
		}
		if configs.ProjectID == "" {
			pterm.Error.Println("GCP project ID is required")
			return errors.New("missing required GCP project ID")
		}

		localImageName, localTag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			return fmt.Errorf("invalid image format: %w", parseErr)
		}
		if localTag == "" {
			localTag = "latest"
		}

		fullGcrImage := fmt.Sprintf("gcr.io/%s/%s:%s", configs.ProjectID, localImageName, localTag)

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

		pterm.Info.Println("Starting GCR build...")
		if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
			pterm.Error.Println("Build failed:", err)
			return err
		}
		pterm.Success.Println("Build completed successfully.")

		pterm.Info.Println("Starting scan...")
		scanErr := docker.Scout(fullGcrImage, configs.SarifFile)
		if scanErr != nil {
			pterm.Error.Println("Scan failed:", scanErr)
		} else {
			pterm.Success.Println("Scan completed successfully.")
		}
		if scanErr != nil {
			return fmt.Errorf("GCR provisioning failed due to previous errors")
		}

		pushImage := fullGcrImage

		if !configs.ConfirmAfterPush {
			pterm.Info.Println("Press Enter to continue...")
			_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
		}

		if configs.ConfirmAfterPush {
			pterm.Info.Printf("Pushing image %s to GCR...\n", pushImage)
			if err := docker.PushImageToGCR(configs.ProjectID, localImageName+":"+localTag); err != nil {
				pterm.Error.Println("Push to GCR failed:", err)
				return err
			}
			pterm.Success.Println("Push to GCR completed successfully.")
		}

		

		if configs.DeleteAfterPush {
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
  # Provide image name[:tag], e.g. "my-app:1.0", along with the project ID:
  smurf sdkr provision-gcr my-app:1.0 -p my-project --file Dockerfile --no-cache \
    --build-arg key1=value1 --build-arg key2=value2 --target my-target --output scan.sarif \
    --yes --delete --platform linux/amd64

  # If you omit the argument, it will read from config and use the parseImage function
  smurf sdkr provision-gcr -p my-project --file Dockerfile
`,
}

func init() {
	provisionGcrCmd.Flags().StringVarP(&configs.ProjectID, "project-id", "p", "", "GCP project ID (required)")

	provisionGcrCmd.Flags().StringVarP(
		&configs.DockerfilePath,
		"file", "f",
		"",
		"Name of the Dockerfile relative to the context directory (default: 'Dockerfile')",
	)
	provisionGcrCmd.Flags().BoolVarP(
		&configs.NoCache,
		"no-cache", "c",
		false,
		"Do not use cache when building the image",
	)
	provisionGcrCmd.Flags().StringArrayVarP(
		&configs.BuildArgs,
		"build-arg", "a",
		[]string{},
		"Set build-time variables (e.g. --build-arg key=value)",
	)
	provisionGcrCmd.Flags().StringVarP(
		&configs.Target,
		"target", "T",
		"",
		"Set the target build stage to build",
	)
	provisionGcrCmd.Flags().StringVarP(
		&configs.Platform,
		"platform", "P",
		"",
		"Set the platform for the image (e.g., linux/amd64)",
	)
	provisionGcrCmd.Flags().StringVar(
		&configs.ContextDir,
		"context",
		"",
		"Build context directory (default: current directory)",
	)
	provisionGcrCmd.Flags().IntVar(
		&configs.BuildTimeout,
		"timeout",
		1500,
		"Build timeout",
	)

	provisionGcrCmd.Flags().StringVarP(
		&configs.SarifFile,
		"output", "o",
		"",
		"Output file for SARIF report",
	)

	provisionGcrCmd.Flags().BoolVarP(
		&configs.ConfirmAfterPush,
		"yes", "y",
		false,
		"Push the image to GCR without confirmation",
	)
	provisionGcrCmd.Flags().BoolVarP(
		&configs.DeleteAfterPush,
		"delete", "d",
		false,
		"Delete the local image after pushing",
	)

	sdkrCmd.AddCommand(provisionGcrCmd)
}
