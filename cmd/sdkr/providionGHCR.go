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

// provisionGHCRCmd sets up the "provision-ghcr" command to build, and push
// Docker images to GitHub Container Registry.
var provisionGHCRCmd = &cobra.Command{
	Use:   "provision-ghcr [IMAGE_NAME[:TAG]]",
	Short: "Build and push a Docker image to GitHub Container Registry",
	Long: `Build and push a Docker image to GitHub Container Registry (GHCR).
	
Authentication:
  - Set USERNAME_GITHUB and TOKEN_GITHUB environment variables
  - OR set them in config file as USERNAME_GITHUB and TOKEN_GITHUB
  - The token must have 'write:packages' scope

Image naming:
  - Images must be in the format: ghcr.io/OWNER/IMAGE_NAME:TAG
  - OWNER can be your username or organization name
  - Example: ghcr.io/my-org/my-app:latest`,

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
		}

		// Validate that image reference includes GHCR registry
		if !strings.HasPrefix(imageRef, "ghcr.io/") {
			pterm.Error.Printfln("GHCR image must start with 'ghcr.io/', got: %s", imageRef)
			pterm.Info.Println("GHCR images must be in the format: ghcr.io/OWNER/IMAGE_NAME:TAG")
			return errors.New("invalid GHCR image format")
		}

		// Check for GitHub credentials
		if os.Getenv("USERNAME_GITHUB") == "" && os.Getenv("TOKEN_GITHUB") == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			envVars := map[string]string{
				"USERNAME_GITHUB": data.Sdkr.GithubUsername,
				"TOKEN_GITHUB":    data.Sdkr.GithubToken,
			}
			if err := configs.ExportEnvironmentVariables(envVars); err != nil {
				return err
			}
		}

		if os.Getenv("USERNAME_GITHUB") == "" || os.Getenv("TOKEN_GITHUB") == "" {
			pterm.Error.Println("GitHub Container Registry credentials are required")
			pterm.Info.Println("You can set them via environment variables:")
			pterm.Info.Println("  export USERNAME_GITHUB=\"your-username\"")
			pterm.Info.Println("  export TOKEN_GITHUB=\"your-github-personal-access-token\"")
			pterm.Info.Println("The token must have 'write:packages' scope")
			pterm.Info.Println("Or set them in your config file as USERNAME_GITHUB and TOKEN_GITHUB")
			return errors.New("missing required GitHub Container Registry credentials")
		}

		// Parse image reference
		localImageName, localTag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			pterm.Error.Printfln("invalid image format: %v", parseErr)
			return fmt.Errorf("invalid image format: %v", parseErr)
		}
		if localImageName == "" {
			pterm.Error.Printfln("invalid image reference")
			return errors.New("invalid image reference")
		}
		if localTag == "" {
			localTag = "latest"
		}

		fullImageName := fmt.Sprintf("%s:%s", localImageName, localTag)

		// Login to GHCR
		// pterm.Info.Println("Logging in to GitHub Container Registry...")
		// loginOpts := docker.LoginOptions{
		// 	Registry: "ghcr.io",
		// 	Username: os.Getenv("USERNAME_GITHUB"),
		// 	Password: os.Getenv("TOKEN_GITHUB"),
		// }
		// if err := docker.Login(loginOpts); err != nil {
		// 	pterm.Error.Println("GHCR login failed:", err)
		// 	return fmt.Errorf("GHCR login failed: %v", err)
		// }
		// pterm.Success.Println("Successfully logged in to GitHub Container Registry")

		// Prepare build arguments
		buildArgsMap := make(map[string]string)
		for _, arg := range configs.BuildArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		// Set context and Dockerfile paths
		if configs.ContextDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				pterm.Error.Printfln("Failed to get current working directory: %v", err)
				return fmt.Errorf("failed to get current working directory: %v", err)
			}
			configs.ContextDir = wd
		}

		if configs.DockerfilePath == "" {
			configs.DockerfilePath = filepath.Join(configs.ContextDir, "Dockerfile")
		} else {
			configs.DockerfilePath = filepath.Join(configs.ContextDir, configs.DockerfilePath)
		}

		// Build options
		buildOpts := docker.BuildOptions{
			DockerfilePath: configs.DockerfilePath,
			NoCache:        configs.NoCache,
			BuildArgs:      buildArgsMap,
			Target:         configs.Target,
			Platform:       configs.Platform,
			Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
			ContextDir:     configs.ContextDir,
		}

		// Build image
		pterm.Info.Println("Starting build...")
		if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
			return err
		}
		pterm.Success.Println("Build completed successfully.")

		// Push image to GHCR
		pterm.Info.Printf("Pushing image %s to GitHub Container Registry...\n", fullImageName)
		pushOpts := docker.PushOptions{
			ImageName: fullImageName,
			Timeout:   1000 * time.Second,
		}
		if err := docker.PushToGHCR(pushOpts); err != nil {
			pterm.Error.Println("Push to GHCR failed:", err)
			return err
		}
		pterm.Success.Printf("Successfully pushed %s to GitHub Container Registry\n", fullImageName)

		// Clean up local image if requested
		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullImageName)
			if err := docker.RemoveImage(fullImageName); err != nil {
				pterm.Warning.Printf("Failed to delete local image: %v\n", err)
				// Don't fail the entire process if delete fails
			} else {
				pterm.Success.Println("Successfully deleted local image:", fullImageName)
			}
		}

		pterm.Success.Println("GHCR provisioning completed successfully.")
		return nil
	},
	Example: `
  # Push to GHCR with full image reference
  smurf sdkr provision-ghcr ghcr.io/my-org/my-image:latest

  # Push with specific tag and build options
  smurf sdkr provision-ghcr ghcr.io/my-username/my-app:v1.0.0 \
    --context . --file Dockerfile --no-cache \
    --build-arg ENV=production --platform linux/amd64 \
    --delete

  # Using environment variables for auth
  export USERNAME_GITHUB="my-username"
  export TOKEN_GITHUB="ghp_yourPersonalAccessToken"
  smurf sdkr provision-ghcr ghcr.io/my-org/my-app:latest

  # Read image name from config file
  smurf sdkr provision-ghcr --delete
`,
}

func init() {
	provisionGHCRCmd.Flags().StringVarP(
		&configs.DockerfilePath,
		"file", "f",
		"",
		"Dockerfile path relative to the context directory (default: 'Dockerfile')",
	)
	provisionGHCRCmd.Flags().BoolVar(
		&configs.NoCache,
		"no-cache",
		false,
		"Do not use cache when building the image",
	)
	provisionGHCRCmd.Flags().StringArrayVar(
		&configs.BuildArgs,
		"build-arg",
		[]string{},
		"Set build-time variables (e.g. --build-arg key=value)",
	)
	provisionGHCRCmd.Flags().StringVar(
		&configs.Target,
		"target",
		"",
		"Set the target build stage to build",
	)
	provisionGHCRCmd.Flags().StringVar(
		&configs.Platform,
		"platform",
		"",
		"Set the platform for the image (e.g., linux/amd64)",
	)
	provisionGHCRCmd.Flags().IntVar(
		&configs.BuildTimeout,
		"timeout",
		1500,
		"Build timeout in seconds",
	)
	provisionGHCRCmd.Flags().StringVar(
		&configs.ContextDir,
		"context",
		"",
		"Build context directory (default: current directory)",
	)
	provisionGHCRCmd.Flags().BoolVarP(
		&configs.ConfirmAfterPush,
		"yes", "y",
		false,
		"Push the image without confirmation",
	)
	provisionGHCRCmd.Flags().BoolVarP(
		&configs.DeleteAfterPush,
		"delete", "d",
		false,
		"Delete the local image after pushing",
	)

	sdkrCmd.AddCommand(provisionGHCRCmd)
}
