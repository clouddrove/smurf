package docker

import (
	"strings"

	"github.com/clouddrove/smurf/internal/docker"
	"github.com/spf13/cobra"
)

var (
	dockerfilePath string
	noCache        bool
	buildArgs      []string
	target         string
	platform       string 
)

var buildCmd = &cobra.Command{
	Use:   "build [IMAGE_NAME] [TAG]",
	Short: "Build a Docker image with the given name and tag.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		buildArgsMap := make(map[string]string)
		for _, arg := range buildArgs {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		opts := docker.BuildOptions{
			DockerfilePath: dockerfilePath,
			NoCache:        noCache,
			BuildArgs:      buildArgsMap,
			Target:         target,
			Platform:       platform, 
		}

		return docker.Build(args[0], args[1], opts)
	},
}

func init() {
	buildCmd.Flags().StringVarP(&dockerfilePath, "file", "f", "Dockerfile", "Name of the Dockerfile (Default is 'Dockerfile')")
	buildCmd.Flags().BoolVar(&noCache, "no-cache", false, "Do not use cache when building the image")
	buildCmd.Flags().StringArrayVar(&buildArgs, "build-arg", []string{}, "Set build-time variables")
	buildCmd.Flags().StringVar(&target, "target", "", "Set the target build stage to build")
	buildCmd.Flags().StringVar(&platform, "platform", "", "Set the platform for the build (e.g., linux/amd64, linux/arm64)")

	sdkrCmd.AddCommand(buildCmd)
}