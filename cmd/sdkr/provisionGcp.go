package sdkr

import (
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
// and optionally push that image to Google Container Registry or Artifact Registry.
var provisionGcpCmd = &cobra.Command{
	Use:   "provision-gcp [IMAGE_NAME[:TAG]]",
	Short: "Build and push a Docker image to Google Container Registry or Artifact Registry.",
	Long: `Build and push a Docker image to Google Container Registry or Artifact Registry.
Set the GOOGLE_APPLICATION_CREDENTIALS environment variable to the path of your service account JSON key file.

Supports:
- Full Artifact Registry path: us-central1-docker.pkg.dev/PROJECT/REPO/IMAGE:TAG
- Full GCR path: gcr.io/PROJECT/IMAGE:TAG  
- Short form: IMAGE:TAG (automatically uses Artifact Registry)
- Repository form: REPO/IMAGE:TAG (automatically uses Artifact Registry)
`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageRef string
		fmt.Println("-------------------------")
		fmt.Println(args[0])
		if len(args) == 1 {
			imageRef = args[0]
		} else {
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
				pterm.Error.Printfln("image name (with optional tag) must be provided either as an argument or in the config")
				return errors.New("image name (with optional tag) must be provided either as an argument or in the config")
			}
			imageRef = data.Sdkr.ImageName
		}

		if configs.ProjectID == "" {
			configs.ProjectID = strings.Split(imageRef, "/")[1]

		}

		// Parse the image reference to get name and tag
		localImageName, localTag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			pterm.Error.Printf("invalid image format: %v\n", parseErr)
			return fmt.Errorf("invalid image format: %v", parseErr)
		}
		if localTag == "" {
			localTag = "latest"
		}

		// Determine the target registry format
		var fullRegistryImage string
		var registryType string
		var buildImageName string

		// Check if the input is already a full registry path
		if strings.Contains(imageRef, ".pkg.dev") || strings.Contains(imageRef, "gcr.io") {
			// User provided full registry path - use it as-is
			fullRegistryImage = imageRef
			if !strings.Contains(imageRef, ":") {
				fullRegistryImage = imageRef + ":" + localTag
			}

			// Extract build image name (last part after /)
			parts := strings.Split(localImageName, "/")
			buildImageName = parts[len(parts)-1]

			if strings.Contains(imageRef, ".pkg.dev") {
				registryType = "Artifact Registry"
			} else {
				registryType = "Google Container Registry"
			}

			pterm.Info.Printf("Using provided %s path: %s\n", registryType, fullRegistryImage)
		} else {
			// User provided short name - construct full path
			if configs.ProjectID == "" {
				pterm.Error.Println("GCP project ID is required when using short image names")
				return errors.New("missing required GCP project ID")
			}

			if configs.UseGCR {
				// Use legacy GCR format
				fullRegistryImage = fmt.Sprintf("gcr.io/%s/%s:%s", configs.ProjectID, localImageName, localTag)
				registryType = "Google Container Registry"
			} else {
				// Use Artifact Registry format (default)
				fullRegistryImage = fmt.Sprintf("us-central1-docker.pkg.dev/%s/%s:%s", configs.ProjectID, localImageName, localTag)
				registryType = "Artifact Registry"
			}
			buildImageName = localImageName
			pterm.Info.Printf("Using %s: %s\n", registryType, fullRegistryImage)
		}

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

		pterm.Info.Println("Starting Docker build...")

		// Build with simple local image name
		localImageRef := buildImageName + ":" + localTag

		if err := docker.Build(buildImageName, localTag, buildOpts); err != nil {
			return err
		}

		// Tag the local image with the full registry path before pushing
		pterm.Info.Printf("Tagging image for %s...\n", registryType)

		tagOpts := docker.TagOptions{
			Source: localImageRef,
			Target: fullRegistryImage,
		}
		if err := docker.TagImage(tagOpts); err != nil {
			pterm.Error.Printf("Failed to tag image: %v\n", err)
			return fmt.Errorf("failed to tag image: %w", err)
		}

		pterm.Info.Printf("Pushing image %s to %s...\n", fullRegistryImage, registryType)

		// Push using the full registry image reference
		if err := docker.PushImageToGCR(configs.ProjectID, fullRegistryImage); err != nil {
			return err
		}

		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local images...\n")

			// Delete the tagged registry image
			if err := docker.RemoveImage(fullRegistryImage); err != nil {
				pterm.Warning.Printf("Failed to delete tagged image %s: %v\n", fullRegistryImage, err)
			}

			// Delete the original local image
			if err := docker.RemoveImage(localImageRef); err != nil {
				pterm.Warning.Printf("Failed to delete local image %s: %v\n", localImageRef, err)
			}

			pterm.Success.Println("Successfully cleaned up local images")
		}

		// Generate and display the registry URL
		var registryURL string
		if strings.Contains(fullRegistryImage, "gcr.io") {
			registryURL = fmt.Sprintf("https://console.cloud.google.com/gcr/images/%s?project=%s",
				strings.ReplaceAll(strings.TrimPrefix(fullRegistryImage, "gcr.io/"), ":", "%3A"), configs.ProjectID)
		} else if strings.Contains(fullRegistryImage, ".pkg.dev") {
			// Artifact Registry console link
			parts := strings.Split(strings.TrimPrefix(fullRegistryImage, "us-central1-docker.pkg.dev/"), "/")
			if len(parts) >= 2 {
				project := parts[0]
				repository := parts[1]
				imageWithTag := strings.Join(parts[2:], "/")
				imageNameOnly := strings.Split(imageWithTag, ":")[0]
				registryURL = fmt.Sprintf("https://console.cloud.google.com/artifacts/docker/%s/us-central1/%s/%s?project=%s",
					project, repository, imageNameOnly, project)
			}
		}

		pterm.Success.Printf("%s provisioning completed successfully.\n", registryType)
		if registryURL != "" {
			pterm.Info.Printf("View your image in Google Cloud Console: %s\n", registryURL)
		}
		pterm.Info.Printf("Image reference: %s\n", fullRegistryImage)

		return nil
	},
	Example: `
  # Build and push using full Artifact Registry path
  smurf sdkr provision-gcp us-central1-docker.pkg.dev/my-project/smurf-test/smurfimage/smurfab:v1

  # Build and push using full GCR path  
  smurf sdkr provision-gcp gcr.io/my-project/myapp:v1.0

  # Build and push with short name (auto Artifact Registry)
  smurf sdkr provision-gcp myapp:v1.0 -p my-project

  # Build and push with repository path (auto Artifact Registry)
  smurf sdkr provision-gcp my-repo/myapp:v1.0 -p my-project

  # Build and push with short name to GCR
  smurf sdkr provision-gcp myapp:v1.0 -p my-project --use-gcr

  # With additional options
  smurf sdkr provision-gcp myapp:v1.0 -p my-project --file Dockerfile --no-cache \
    --build-arg key1=value1 --build-arg key2=value2 --target my-target \
    --delete --platform linux/amd64
`,
}

func init() {
	provisionGcpCmd.Flags().StringVarP(&configs.ProjectID, "project-id", "p", "", "GCP project ID (required for short image names)")

	provisionGcpCmd.Flags().BoolVar(
		&configs.UseGCR,
		"use-gcr",
		false,
		"Use legacy Google Container Registry (gcr.io) instead of Artifact Registry",
	)

	provisionGcpCmd.Flags().StringVarP(
		&configs.DockerfilePath,
		"file", "f",
		"",
		"Name of the Dockerfile relative to the context directory (default: 'Dockerfile')",
	)
	provisionGcpCmd.Flags().BoolVarP(
		&configs.NoCache,
		"no-cache", "c",
		false,
		"Do not use cache when building the image",
	)
	provisionGcpCmd.Flags().StringArrayVarP(
		&configs.BuildArgs,
		"build-arg", "a",
		[]string{},
		"Set build-time variables (e.g. --build-arg key=value)",
	)
	provisionGcpCmd.Flags().StringVarP(
		&configs.Target,
		"target", "T",
		"",
		"Set the target build stage to build",
	)
	provisionGcpCmd.Flags().StringVarP(
		&configs.Platform,
		"platform", "P",
		"",
		"Set the platform for the image (e.g., linux/amd64)",
	)
	provisionGcpCmd.Flags().StringVar(
		&configs.ContextDir,
		"context",
		"",
		"Build context directory (default: current directory)",
	)
	provisionGcpCmd.Flags().IntVar(
		&configs.BuildTimeout,
		"timeout",
		1500,
		"Build timeout in seconds",
	)

	provisionGcpCmd.Flags().StringVarP(
		&configs.SarifFile,
		"output", "o",
		"",
		"Output file for SARIF report",
	)

	provisionGcpCmd.Flags().BoolVarP(
		&configs.ConfirmAfterPush,
		"yes", "y",
		false,
		"Push the image to registry without confirmation",
	)
	provisionGcpCmd.Flags().BoolVarP(
		&configs.DeleteAfterPush,
		"delete", "d",
		false,
		"Delete the local image after pushing",
	)

	sdkrCmd.AddCommand(provisionGcpCmd)
}
