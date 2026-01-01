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

// provisionGHCRCmd defines the "provision-ghcr" CLI command for GHCR operations.
var provisionGHCRCmd = &cobra.Command{
	Use:   "provision-ghcr [IMAGE_NAME[:TAG]]",
	Short: "Build and push a Docker image to GitHub Container Registry",
	Long: `Build and push a Docker image to GitHub Container Registry (GHCR).

Authentication:
  - Set USERNAME_GITHUB and TOKEN_GITHUB environment variables
  - OR define them in config file (USERNAME_GITHUB, TOKEN_GITHUB)
  - The token must have 'write:packages' scope

Image format:
  ghcr.io/OWNER/IMAGE_NAME:TAG
Example: ghcr.io/my-org/my-app:latest`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE:         runProvisionGHCR,
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
	provisionGHCRCmd.Flags().StringVarP(&configs.DockerfilePath, "file", "f", "", "Path to Dockerfile (default: Dockerfile)")
	provisionGHCRCmd.Flags().BoolVar(&configs.NoCache, "no-cache", false, "Disable build cache")
	provisionGHCRCmd.Flags().StringArrayVar(&configs.BuildArgs, "build-arg", []string{}, "Build-time variables (e.g. --build-arg key=value)")
	provisionGHCRCmd.Flags().StringVar(&configs.Target, "target", "", "Target build stage")
	provisionGHCRCmd.Flags().StringVar(&configs.Platform, "platform", "", "Platform (e.g. linux/amd64)")
	provisionGHCRCmd.Flags().IntVar(&configs.BuildTimeout, "timeout", 1500, "Build timeout in seconds")
	provisionGHCRCmd.Flags().StringVar(&configs.ContextDir, "context", "", "Build context (default: current directory)")
	provisionGHCRCmd.Flags().BoolVarP(&configs.ConfirmAfterPush, "yes", "y", false, "Push without confirmation")
	provisionGHCRCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete local image after push")
	provisionGHCRCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	sdkrCmd.AddCommand(provisionGHCRCmd)
}

func runProvisionGHCR(cmd *cobra.Command, args []string) error {
	cfg, err := configs.LoadConfig(configs.FileName)
	if err != nil {
		return err
	}

	imageRef := resolveImageRef(args, cfg)
	if imageRef == "" {
		return errors.New("image reference not provided")
	}

	if err := validateGHCRImage(imageRef); err != nil {
		return err
	}

	if err := ensureGHCRAuth(cfg); err != nil {
		return err
	}

	imageName, tag, err := configs.ParseImage(imageRef)
	if err != nil {
		return fmt.Errorf("invalid image reference: %v", err)
	}
	if tag == "" {
		tag = "latest"
	}

	fullImage := fmt.Sprintf("%s:%s", imageName, tag)
	pterm.Info.Printfln("Preparing to build and push image: %s", fullImage)

	buildOpts, err := prepareBuildOptions()
	if err != nil {
		return err
	}

	if err := docker.Build(imageName, tag, buildOpts, useAI); err != nil {
		return fmt.Errorf("build failed: %v", err)
	}
	pterm.Success.Println("✅ Build completed successfully.")

	if err := pushToGHCR(fullImage); err != nil {
		return err
	}

	if configs.DeleteAfterPush {
		cleanupLocalImage(fullImage)
	}

	pterm.Success.Println("🚀 GHCR provisioning completed successfully.")
	return nil
}

func resolveImageRef(args []string, cfg *configs.Config) string {
	if len(args) > 0 {
		return args[0]
	}
	if cfg.Sdkr.ImageName != "" {
		return cfg.Sdkr.ImageName
	}
	pterm.Error.Println("Image name must be provided either as argument or in config file.")
	return ""
}

func validateGHCRImage(image string) error {
	if !strings.HasPrefix(image, "ghcr.io/") {
		pterm.Error.Printfln("Invalid GHCR image format: %s", image)
		pterm.Info.Println("Expected format: ghcr.io/OWNER/IMAGE_NAME:TAG")
		return errors.New("invalid GHCR image format")
	}
	return nil
}

func ensureGHCRAuth(cfg *configs.Config) error {
	username := os.Getenv("USERNAME_GITHUB")
	token := os.Getenv("TOKEN_GITHUB")

	if username == "" || token == "" {
		envVars := map[string]string{
			"USERNAME_GITHUB": cfg.Sdkr.GithubUsername,
			"TOKEN_GITHUB":    cfg.Sdkr.GithubToken,
		}
		if err := configs.ExportEnvironmentVariables(envVars); err != nil {
			return err
		}
		username = envVars["USERNAME_GITHUB"]
		token = envVars["TOKEN_GITHUB"]
	}

	if username == "" || token == "" {
		pterm.Error.Println("GitHub Container Registry credentials missing.")
		pterm.Info.Println("Set using environment variables:")
		pterm.Info.Println("  export USERNAME_GITHUB=\"your-username\"")
		pterm.Info.Println("  export TOKEN_GITHUB=\"your-github-personal-access-token\"")
		return errors.New("missing GHCR credentials")
	}
	return nil
}

func prepareBuildOptions() (docker.BuildOptions, error) {
	if configs.ContextDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return docker.BuildOptions{}, fmt.Errorf("failed to get working directory: %v", err)
		}
		configs.ContextDir = wd
	}

	if configs.DockerfilePath == "" {
		configs.DockerfilePath = filepath.Join(configs.ContextDir, "Dockerfile")
	} else {
		configs.DockerfilePath = filepath.Join(configs.ContextDir, configs.DockerfilePath)
	}

	buildArgsMap := make(map[string]string)
	for _, arg := range configs.BuildArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			buildArgsMap[parts[0]] = parts[1]
		}
	}

	return docker.BuildOptions{
		DockerfilePath: configs.DockerfilePath,
		NoCache:        configs.NoCache,
		BuildArgs:      buildArgsMap,
		Target:         configs.Target,
		Platform:       configs.Platform,
		Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
		ContextDir:     configs.ContextDir,
	}, nil
}

func pushToGHCR(fullImage string) error {
	pterm.Info.Printf("📦 Pushing image %s to GitHub Container Registry...\n", fullImage)
	pushOpts := docker.PushOptions{
		ImageName: fullImage,
		Timeout:   1000 * time.Second,
	}
	if err := docker.PushToGHCR(pushOpts, useAI); err != nil {
		pterm.Error.Printfln("Push failed: %v", err)
		return err
	}
	pterm.Success.Printfln("✅ Successfully pushed %s to GHCR", fullImage)
	return nil
}

func cleanupLocalImage(fullImage string) {
	pterm.Info.Printf("🧹 Deleting local image %s...\n", fullImage)
	if err := docker.RemoveImage(fullImage, useAI); err != nil {
		pterm.Warning.Printfln("Failed to delete local image: %v", err)
	} else {
		pterm.Success.Println("Local image deleted successfully.")
	}
}
