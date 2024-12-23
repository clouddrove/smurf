package docker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)


var buildCmd = &cobra.Command{
	Use:   "build [IMAGE[:TAG]]",
	Short: "Build a Docker image with the given name and tag.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageName, tag string

		

		if len(args) >= 1 {
			var err error
			imageName, tag, err = parseImage(args[0])
			if err != nil {
				return fmt.Errorf("invalid image format: %w", err)
			}
		}

		if imageName == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if len(args) < 1 && data.Sdkr.ImageName != "" {
				imageName, tag, err = parseImage(data.Sdkr.ImageName)
				if err != nil {
					return fmt.Errorf("invalid image format in config: %w", err)
				}
				if tag == "" {
					tag = "latest"
				}
			}

			if imageName == "" {
				pterm.Warning.Println("No image name provided. Please provide an image name as an argument or in the config.")
				return errors.New("image name must be provided either as an argument or in the config")
			}
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
				return fmt.Errorf("failed to get current working directory: %w", err)
			}
			configs.ContextDir = wd
		}

		if configs.DockerfilePath == "" {
			configs.DockerfilePath = filepath.Join(configs.ContextDir, "Dockerfile")
		} else {
			configs.DockerfilePath = filepath.Join(configs.ContextDir, configs.DockerfilePath)
		}

		if _, err := os.Stat(configs.DockerfilePath); os.IsNotExist(err) {
			return fmt.Errorf(color.RedString("Dockerfile not found at %s", configs.DockerfilePath))
		}

		opts := docker.BuildOptions{
			ContextDir:     configs.ContextDir,
			DockerfilePath: configs.DockerfilePath,
			NoCache:        configs.NoCache,
			BuildArgs:      buildArgsMap,
			Target:         configs.Target,
			Platform:       configs.Platform,
			Timeout:        configs.BuildTimeout,
		}

		err := docker.Build(imageName, tag, opts)
		if err != nil {
			return fmt.Errorf(color.RedString("Docker build failed: %v", err))
		}
		return nil
	},
	Example: `
smurf sdkr build my-image:v1
smurf sdkr build my-image:v1 --file Dockerfile --context ./build-context --no-cache --build-arg key1=value1 --build-arg key2=value2 --target my-target --platform linux/amd64 --timeout 10m
smurf sdkr build
# In the last example, it will read "image:v1" from config and use the parsed image name and tag
`,
}

func init() {
	buildCmd.Flags().StringVarP(&configs.DockerfilePath, "file", "f", "", "Path to Dockerfile relative to context directory")
	buildCmd.Flags().StringVar(&configs.ContextDir, "context", "", "Build context directory (default: current directory)")
	buildCmd.Flags().BoolVar(&configs.NoCache, "no-cache", false, "Do not use cache when building the image")
	buildCmd.Flags().StringArrayVar(&configs.BuildArgs, "build-arg", []string{}, "Set build-time variables")
	buildCmd.Flags().StringVar(&configs.Target, "target", "", "Set the target build stage to build")
	buildCmd.Flags().StringVar(&configs.Platform, "platform", "", "Set the platform for the build (e.g., linux/amd64, linux/arm64)")
	buildCmd.Flags().DurationVar(&configs.BuildTimeout, "timeout", 25*time.Minute, "Set the build timeout")

	sdkrCmd.AddCommand(buildCmd)
}


func parseImage(image string) (string, string, error) {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return image, "latest", nil 
}