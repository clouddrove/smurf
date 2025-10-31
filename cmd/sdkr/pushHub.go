package sdkr

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// pushHubCmd defines the "hub" command, which pushes Docker images to Docker Hub.
// It supports both command-line arguments and config file values for the image reference,
// as well as environment variables or config-defined credentials for authentication.
// Optionally, the local image can be removed after a successful push.
var pushHubCmd = &cobra.Command{
	Use:   "hub [IMAGE_NAME[:TAG]]",
	Short: "Push Docker images to Docker Hub",
	Long: `
Push Docker images to Docker Hub.
Export DOCKER_USERNAME and DOCKER_PASSWORD as environment variables for Docker Hub authentication, for example:
  export DOCKER_USERNAME="your-username"
  export DOCKER_PASSWORD="your-password"`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageRef string
		var envVars map[string]string

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

			envVars = map[string]string{
				"DOCKER_USERNAME": data.Sdkr.DockerUsername,
				"DOCKER_PASSWORD": data.Sdkr.DockerPassword,
			}
			if err := configs.ExportEnvironmentVariables(envVars); err != nil {
				pterm.Error.Println("Error exporting Docker Hub credentials:", err)
				return err
			}
		}

		if os.Getenv("DOCKER_PASSWORD") == "" || os.Getenv("DOCKER_PASSWORD") == "" {
			pterm.Error.Println("ired")
			return errors.New("missing required Docker Hub credentials")
		}

		repoName, tag, parseErr := configs.ParseImage(imageRef)
		if parseErr != nil {
			return parseErr
		}
		if repoName == "" {
			pterm.Error.Println("invalid image reference")
			return errors.New("invalid image reference")
		}
		if tag == "" {
			tag = "latest"
		}

		fullImageName := fmt.Sprintf("%s:%s", repoName, tag)
		pterm.Info.Printf("Pushing image %s to Docker Hub...\n", fullImageName)

		opts := docker.PushOptions{
			ImageName: fullImageName,
			Timeout:   time.Duration(configs.BuildTimeout) * time.Second,
		}
		if err := docker.PushImage(opts); err != nil {
			pterm.Error.Println("Failed to push image to Docker Hub:", err)
			return err
		}
		pterm.Success.Println("Successfully pushed image to Docker Hub:", fullImageName)

		if configs.DeleteAfterPush {
			pterm.Info.Printf("Deleting local image %s...\n", fullImageName)
			if err := docker.RemoveImage(fullImageName); err != nil {
				return err
			}
			pterm.Success.Println("Successfully deleted local image:", fullImageName)
		}

		return nil
	},
	Example: `
  smurf sdkr push hub myapp:v1
  smurf sdkr push hub myapp:v1 --delete
`,
}

func init() {
	pushHubCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	pushHubCmd.Flags().IntVar(&configs.BuildTimeout, "timeout", 1500, "Timeout for the push operation in minutes")
	pushCmd.AddCommand(pushHubCmd)
}
