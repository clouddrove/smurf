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

var useAI bool

// buildCmd defines the "build" subcommand for creating Docker images.
// It reads image details from either command-line arguments or a config file,
// supports setting build context, specifying a Dockerfile, and passing build-time variables.
// Usage examples are included within the command definition below, showcasing various ways
// to override defaults (e.g., no-cache, target stage, platform, and timeout).
var buildCmd = &cobra.Command{
	Use:          "build [IMAGE[:TAG]]",
	Short:        "Build a Docker image with the given name and tag.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var imageName, tag string

		if len(args) >= 1 {
			var err error
			imageName, tag, err = configs.ParseImage(args[0])
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
				imageName, tag, err = configs.ParseImage(data.Sdkr.ImageName)
				if err != nil {
					return fmt.Errorf("invalid image format in config: %v", err)
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
			return fmt.Errorf("dockerfile not found at %v", configs.DockerfilePath)
		}

		// In the RunE function, when creating the opts
		opts := docker.BuildOptions{
			ContextDir:     configs.ContextDir,
			DockerfilePath: configs.DockerfilePath,
			NoCache:        configs.NoCache,
			BuildArgs:      buildArgsMap,
			Target:         configs.Target,
			Platform:       configs.Platform,
			Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
			BuildKit:       configs.BuildKit,
		}

		err := docker.Build(imageName, tag, opts, useAI)
		if err != nil {
			return err
		}
		return nil
	},
	Example: `
smurf sdkr build my-image:v1
smurf sdkr build my-image:v1 --file Dockerfile --context ./build-context --no-cache --build-arg key1=value1 --build-arg key2=value2 --target my-target --platform linux/amd64 --timeout 400
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
	buildCmd.Flags().IntVar(&configs.BuildTimeout, "timeout", 1500, "Set the build timeout")
	buildCmd.Flags().BoolVar(&configs.BuildKit, "buildkit", false, "Enable BuildKit for advanced Dockerfile features")
	buildCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	sdkrCmd.AddCommand(buildCmd)
}
