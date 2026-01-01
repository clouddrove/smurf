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

// provisionHubCmd sets up the "provision-hub" command to build, scan, and optionally push
// Docker images to Docker Hub. It also supports setting Docker Hub credentials via environment
// variables or config, uses build arguments, and allows automated cleanup of local images
// after a successful push.
var provisionHubCmd = &cobra.Command{
	Use:   "provision-hub [IMAGE_NAME[:TAG]]",
	Short: "Build, scan, and push a Docker image .",
	Long: `Build, scan, and push a Docker image to Docker Hub.
	Set DOCKER_USERNAME and DOCKER_PASSWORD environment variables for Docker Hub authentication, for example:
  	export DOCKER_USERNAME="your-username"
  	export DOCKER_PASSWORD="your-password"`,
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

		if os.Getenv("DOCKER_USERNAME") == "" && os.Getenv("DOCKER_PASSWORD") == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			envVars := map[string]string{
				"DOCKER_USERNAME": data.Sdkr.DockerUsername,
				"DOCKER_PASSWORD": data.Sdkr.DockerPassword,
			}
			if err := configs.ExportEnvironmentVariables(envVars); err != nil {
				return err
			}
		}

		if os.Getenv("DOCKER_USERNAME") == "" || os.Getenv("DOCKER_PASSWORD") == "" {
			fmt.Println("error : ", os.Getenv("DOCKER_USERNAME"), "&&", os.Getenv("DOCKER_PASSWORD"))
			pterm.Error.Println("Docker Hub credentials are required")
			return errors.New("missing required Docker Hub credentials")
		}

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

		buildOpts := docker.BuildOptions{
			DockerfilePath: configs.DockerfilePath,
			NoCache:        configs.NoCache,
			BuildArgs:      buildArgsMap,
			Target:         configs.Target,
			Platform:       configs.Platform,
			Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
			ContextDir:     configs.ContextDir,
		}

		pterm.Info.Println("Starting build...")
		if err := docker.Build(localImageName, localTag, buildOpts, useAI); err != nil {
			return err
		}
		pterm.Success.Println("Build completed successfully.")
		/*
			pterm.Info.Println("Starting scan with Trivy...")
			scanErr := docker.Trivy(fullImageName)
			if scanErr != nil {
				return scanErr
			}
		*/
		/*
			if !configs.ConfirmAfterPush {
				pterm.Info.Println("Press Enter to continue...")
				_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
			}*/

		pterm.Info.Printf("Pushing image %s...\n", fullImageName)
		pushOpts := docker.PushOptions{
			ImageName: fullImageName,
			Timeout:   1000000000000,
		}
		if err := docker.PushImage(pushOpts, useAI); err != nil {
			pterm.Error.Println("Push failed:", err)
			return err
		}

		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullImageName)
			if err := docker.RemoveImage(fullImageName, useAI); err != nil {
				pterm.Error.Println("Failed to delete local image:", err)
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", fullImageName)
		}

		pterm.Success.Println("Provisioning completed successfully.")
		return nil
	},
	Example: `
  # Provide "myuser/myimage:latest" as an argument
  smurf sdkr provision-hub myuser/myimage:latest --context . --file Dockerfile --no-cache \
    --build-arg key1=value1 --build-arg key2=value2 --target my-target --platform linux/amd64 \
    --yes --delete

  # If you omit the argument, it will read from config and rely on "image_name" from there
  smurf sdkr provision-hub --yes --delete
`,
}

func init() {
	provisionHubCmd.Flags().StringVarP(
		&configs.DockerfilePath,
		"file", "f",
		"",
		"Dockerfile path relative to the context directory (default: 'Dockerfile')",
	)
	provisionHubCmd.Flags().BoolVar(
		&configs.NoCache,
		"no-cache",
		false,
		"Do not use cache when building the image",
	)
	provisionHubCmd.Flags().StringArrayVar(
		&configs.BuildArgs,
		"build-arg",
		[]string{},
		"Set build-time variables (e.g. --build-arg key=value)",
	)
	provisionHubCmd.Flags().StringVar(
		&configs.Target,
		"target",
		"",
		"Set the target build stage to build",
	)
	provisionHubCmd.Flags().StringVar(
		&configs.Platform,
		"platform",
		"",
		"Set the platform for the image (e.g., linux/amd64)",
	)
	provisionHubCmd.Flags().IntVar(
		&configs.BuildTimeout,
		"timeout",
		1500,
		"Build timeout",
	)
	provisionHubCmd.Flags().StringVar(
		&configs.ContextDir,
		"context",
		"",
		"Build context directory (default: current directory)",
	)
	provisionHubCmd.Flags().BoolVarP(
		&configs.ConfirmAfterPush,
		"yes", "y",
		false,
		"Push the image without confirmation",
	)
	provisionHubCmd.Flags().BoolVarP(
		&configs.DeleteAfterPush,
		"delete", "d",
		false,
		"Delete the local image after pushing",
	)
	provisionHubCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	sdkrCmd.AddCommand(provisionHubCmd)
}
