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

// Constants for registry types and URLs
const (
	// Registry domains
	ArtifactRegistryDomain = ".pkg.dev"
	GCRDomain              = "gcr.io"

	// Registry types
	ArtifactRegistryType = "Artifact Registry"
	GCRRegistryType      = "Google Container Registry"

	// Default values
	DefaultTag             = "latest"
	DefaultTimeout         = 1500
	ArtifactRegistryFormat = "us-central1-docker.pkg.dev/%s/%s:%s"
	GCRFormat              = "gcr.io/%s/%s:%s"
)

// ImageRegistry handles image registry operations and parsing
type ImageRegistry struct {
	ProjectID       string
	UseGCR          bool
	DeleteAfterPush bool
}

// ImageReference represents a parsed image reference
type ImageReference struct {
	FullPath       string
	LocalName      string
	LocalTag       string
	RegistryType   string
	BuildImageName string
	RegistryURL    string
}

// NewImageRegistry creates a new ImageRegistry instance
func NewImageRegistry(projectID string, useGCR, deleteAfterPush bool) *ImageRegistry {
	return &ImageRegistry{
		ProjectID:       projectID,
		UseGCR:          useGCR,
		DeleteAfterPush: deleteAfterPush,
	}
}

// ParseImageReference parses and validates image reference
func (ir *ImageRegistry) ParseImageReference(imageRef string) (*ImageReference, error) {
	localImageName, localTag, parseErr := configs.ParseImage(imageRef)
	if parseErr != nil {
		return nil, fmt.Errorf("invalid image format: %v", parseErr)
	}

	if localTag == "" {
		localTag = DefaultTag
	}

	// Check if the input is already a full registry path
	if strings.Contains(imageRef, ArtifactRegistryDomain) || strings.Contains(imageRef, GCRDomain) {
		return ir.parseFullRegistryPath(imageRef, localImageName, localTag)
	}

	return ir.parseShortImageName(imageRef, localImageName, localTag)
}

// parseFullRegistryPath handles full registry paths
func (ir *ImageRegistry) parseFullRegistryPath(imageRef, localImageName, localTag string) (*ImageReference, error) {
	fullRegistryImage := imageRef
	if !strings.Contains(imageRef, ":") {
		fullRegistryImage = imageRef + ":" + localTag
	}

	// Extract build image name (last part after /)
	parts := strings.Split(localImageName, "/")
	buildImageName := parts[len(parts)-1]

	var registryType string
	if strings.Contains(imageRef, ArtifactRegistryDomain) {
		registryType = ArtifactRegistryType
	} else {
		registryType = GCRRegistryType
	}

	pterm.Info.Printf("Using provided %s path: %s\n", registryType, fullRegistryImage)

	return &ImageReference{
		FullPath:       fullRegistryImage,
		LocalName:      localImageName,
		LocalTag:       localTag,
		RegistryType:   registryType,
		BuildImageName: buildImageName,
	}, nil
}

// parseShortImageName handles short image names and constructs full registry paths
func (ir *ImageRegistry) parseShortImageName(imageRef, localImageName, localTag string) (*ImageReference, error) {
	if ir.ProjectID == "" {
		return nil, errors.New("GCP project ID is required when using short image names")
	}

	var fullRegistryImage, registryType string
	if ir.UseGCR {
		// Use legacy GCR format
		fullRegistryImage = fmt.Sprintf(GCRFormat, ir.ProjectID, localImageName, localTag)
		registryType = GCRRegistryType
	} else {
		// Use Artifact Registry format (default)
		fullRegistryImage = fmt.Sprintf(ArtifactRegistryFormat, ir.ProjectID, localImageName, localTag)
		registryType = ArtifactRegistryType
	}

	pterm.Info.Printf("Using %s: %s\n", registryType, fullRegistryImage)

	return &ImageReference{
		FullPath:       fullRegistryImage,
		LocalName:      localImageName,
		LocalTag:       localTag,
		RegistryType:   registryType,
		BuildImageName: localImageName,
	}, nil
}

// GenerateRegistryURL generates the Google Cloud Console URL for the image
func (ir *ImageRegistry) GenerateRegistryURL(imageRef *ImageReference) string {
	if strings.Contains(imageRef.FullPath, GCRDomain) {
		return ir.generateGCRURL(imageRef)
	} else if strings.Contains(imageRef.FullPath, ArtifactRegistryDomain) {
		return ir.generateArtifactRegistryURL(imageRef)
	}
	return ""
}

func (ir *ImageRegistry) generateGCRURL(imageRef *ImageReference) string {
	return fmt.Sprintf("https://console.cloud.google.com/gcr/images/%s?project=%s",
		strings.ReplaceAll(strings.TrimPrefix(imageRef.FullPath, GCRDomain+"/"), ":", "%3A"), ir.ProjectID)
}

func (ir *ImageRegistry) generateArtifactRegistryURL(imageRef *ImageReference) string {
	parts := strings.Split(strings.TrimPrefix(imageRef.FullPath, "us-central1-docker.pkg.dev/"), "/")
	if len(parts) >= 2 {
		project := parts[0]
		repository := parts[1]
		imageWithTag := strings.Join(parts[2:], "/")
		imageNameOnly := strings.Split(imageWithTag, ":")[0]
		return fmt.Sprintf("https://console.cloud.google.com/artifacts/docker/%s/us-central1/%s/%s?project=%s",
			project, repository, imageNameOnly, project)
	}
	return ""
}

// BuildConfig handles Docker build configuration
type BuildConfig struct {
	ContextDir     string
	DockerfilePath string
	NoCache        bool
	BuildArgs      []string
	Target         string
	Platform       string
	Timeout        int
}

// NewBuildConfig creates a new BuildConfig with defaults
func NewBuildConfig() *BuildConfig {
	wd, err := os.Getwd()
	if err != nil {
		pterm.Warning.Printf("Failed to get current working directory: %v\n", err)
		wd = "."
	}

	return &BuildConfig{
		ContextDir: wd,
		Timeout:    DefaultTimeout,
	}
}

// PrepareBuildOptions prepares docker build options from configuration
func (bc *BuildConfig) PrepareBuildOptions() docker.BuildOptions {
	// Set Dockerfile path
	dockerfilePath := bc.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = filepath.Join(bc.ContextDir, "Dockerfile")
	} else {
		dockerfilePath = filepath.Join(bc.ContextDir, dockerfilePath)
	}

	// Parse build args
	buildArgsMap := make(map[string]string)
	for _, arg := range bc.BuildArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			buildArgsMap[parts[0]] = parts[1]
		}
	}

	return docker.BuildOptions{
		ContextDir:     bc.ContextDir,
		DockerfilePath: dockerfilePath,
		NoCache:        bc.NoCache,
		BuildArgs:      buildArgsMap,
		Target:         bc.Target,
		Platform:       bc.Platform,
		Timeout:        time.Duration(bc.Timeout) * time.Second,
	}
}

// loadConfiguration loads configuration from file or environment
func loadConfiguration(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}

	data, err := configs.LoadConfig(configs.FileName)
	if err != nil {
		return "", err
	}

	// Set environment variables if needed
	envVars := make(map[string]string)
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" && data.Sdkr.GoogleApplicationCredentials != "" {
		envVars["GOOGLE_APPLICATION_CREDENTIALS"] = data.Sdkr.GoogleApplicationCredentials
	}

	if len(envVars) > 0 {
		if err := configs.ExportEnvironmentVariables(envVars); err != nil {
			return "", err
		}
	}

	if envVars["GOOGLE_APPLICATION_CREDENTIALS"] == "" {
		return "", errors.New("missing required Google Application Credentials")
	}

	if data.Sdkr.ImageName == "" {
		return "", errors.New("image name (with optional tag) must be provided either as an argument or in the config")
	}

	return data.Sdkr.ImageName, nil
}

// cleanupImages cleans up local images after push if configured
func cleanupImages(imageRef *ImageReference, registry *ImageRegistry) {
	if !registry.DeleteAfterPush {
		return
	}

	pterm.Info.Printf("Deleting local images...\n")

	localImageRef := imageRef.BuildImageName + ":" + imageRef.LocalTag

	// Delete the tagged registry image
	if err := docker.RemoveImage(imageRef.FullPath, useAI); err != nil {
		pterm.Warning.Printf("Failed to delete tagged image %s: %v\n", imageRef.FullPath, err)
	}

	// Delete the original local image
	if err := docker.RemoveImage(localImageRef, useAI); err != nil {
		pterm.Warning.Printf("Failed to delete local image %s: %v\n", localImageRef, err)
	}

	pterm.Success.Println("Successfully cleaned up local images")
}

// provisionGcpCmd encapsulates the logic to build a Docker image, run a security scan,
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
		// Load configuration
		imageRef, err := loadConfiguration(args)
		if err != nil {
			pterm.Error.Println(err.Error())
			return err
		}

		// Initialize registry and build config
		registry := NewImageRegistry(configs.ProjectID, configs.UseGCR, configs.DeleteAfterPush)
		buildConfig := NewBuildConfig()

		// Parse image reference
		parsedImage, err := registry.ParseImageReference(imageRef)
		if err != nil {
			pterm.Error.Println(err.Error())
			return err
		}

		// Set project ID from image reference if not set
		if configs.ProjectID == "" {
			parts := strings.Split(imageRef, "/")
			if len(parts) > 1 {
				configs.ProjectID = parts[1]
			}
		}

		// Configure build options
		buildConfig.ContextDir = configs.ContextDir
		buildConfig.DockerfilePath = configs.DockerfilePath
		buildConfig.NoCache = configs.NoCache
		buildConfig.BuildArgs = configs.BuildArgs
		buildConfig.Target = configs.Target
		buildConfig.Platform = configs.Platform
		buildConfig.Timeout = configs.BuildTimeout

		buildOpts := buildConfig.PrepareBuildOptions()

		// Build Docker image
		pterm.Info.Println("Starting Docker build...")
		localImageRef := parsedImage.BuildImageName + ":" + parsedImage.LocalTag

		if err := docker.Build(parsedImage.BuildImageName, parsedImage.LocalTag, buildOpts, useAI); err != nil {
			return err
		}

		// Tag image for registry
		pterm.Info.Printf("Tagging image for %s...\n", parsedImage.RegistryType)
		tagOpts := docker.TagOptions{
			Source: localImageRef,
			Target: parsedImage.FullPath,
		}
		if err := docker.TagImage(tagOpts, useAI); err != nil {
			pterm.Error.Printf("Failed to tag image: %v\n", err)
			return fmt.Errorf("failed to tag image: %w", err)
		}

		// Push to registry
		pterm.Info.Printf("Pushing image %s to %s...\n", parsedImage.FullPath, parsedImage.RegistryType)
		if err := docker.PushImageToGCR(configs.ProjectID, parsedImage.FullPath, useAI); err != nil {
			return err
		}

		// Cleanup images if configured
		cleanupImages(parsedImage, registry)

		// Generate and display registry URL
		registryURL := registry.GenerateRegistryURL(parsedImage)

		pterm.Success.Printf("%s provisioning completed successfully.\n", parsedImage.RegistryType)
		if registryURL != "" {
			pterm.Info.Printf("View your image in Google Cloud Console: %s\n", registryURL)
		}
		pterm.Info.Printf("Image reference: %s\n", parsedImage.FullPath)

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
	// Project and registry flags
	provisionGcpCmd.Flags().StringVarP(&configs.ProjectID, "project-id", "p", "", "GCP project ID (required for short image names)")
	provisionGcpCmd.Flags().BoolVar(&configs.UseGCR, "use-gcr", false, "Use legacy Google Container Registry (gcr.io) instead of Artifact Registry")

	// Build configuration flags
	provisionGcpCmd.Flags().StringVarP(&configs.DockerfilePath, "file", "f", "", "Name of the Dockerfile relative to the context directory (default: 'Dockerfile')")
	provisionGcpCmd.Flags().BoolVarP(&configs.NoCache, "no-cache", "c", false, "Do not use cache when building the image")
	provisionGcpCmd.Flags().StringArrayVarP(&configs.BuildArgs, "build-arg", "a", []string{}, "Set build-time variables (e.g. --build-arg key=value)")
	provisionGcpCmd.Flags().StringVarP(&configs.Target, "target", "T", "", "Set the target build stage to build")
	provisionGcpCmd.Flags().StringVarP(&configs.Platform, "platform", "P", "", "Set the platform for the image (e.g., linux/amd64)")
	provisionGcpCmd.Flags().StringVar(&configs.ContextDir, "context", "", "Build context directory (default: current directory)")
	provisionGcpCmd.Flags().IntVar(&configs.BuildTimeout, "timeout", DefaultTimeout, "Build timeout in seconds")

	// Output and behavior flags
	provisionGcpCmd.Flags().StringVarP(&configs.SarifFile, "output", "o", "", "Output file for SARIF report")
	provisionGcpCmd.Flags().BoolVarP(&configs.ConfirmAfterPush, "yes", "y", false, "Push the image to registry without confirmation")
	provisionGcpCmd.Flags().BoolVarP(&configs.DeleteAfterPush, "delete", "d", false, "Delete the local image after pushing")
	provisionGcpCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	sdkrCmd.AddCommand(provisionGcpCmd)
}
