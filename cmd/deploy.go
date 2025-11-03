package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/docker"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:          "deploy",
	Short:        "Deploy is used to perform operation provided smurf.yml file configuration.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1️⃣ Load config
		data, err := configs.LoadConfig(configs.FileName)
		if err != nil {
			return err
		}

		imageName := data.Sdkr.ImageName
		if imageName == "" {
			return fmt.Errorf("no image name provided in smurf.yaml or CLI argument")
		}

		var imageRepo, imageTag string

		// 2️⃣ Handle registry push
		switch {
		case data.Sdkr.AwsECR:
			imageRepo, imageTag, err = handleECRPush(data)
		case data.Sdkr.DockerHub:
			imageRepo, imageTag, err = handleDockerHubPush(data)
		case data.Sdkr.GHCRRepo:
			imageRepo, imageTag, err = handleGHCRPush(data)
		default:
			pterm.Warning.Println("No registry selected (awsECR/dockerHub/ghcrRepo). Skipping image push.")
		}
		if err != nil {
			return err
		}

		// 3️⃣ Helm deployment
		if data.Selm.HelmDeploy {
			if err := handleHelmDeploy(data, imageRepo, imageTag); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(deployCmd)
}

func handleECRPush(data *configs.Config) (string, string, error) {
	pterm.Info.Println("Pushing image to AWS ECR...")

	localImageName, localTag, err := configs.ParseImage(data.Sdkr.ImageName)
	if err != nil {
		return "", "", fmt.Errorf("invalid image name: %v", err)
	}
	if localTag == "" {
		localTag = "latest"
	}

	// Determine context directory
	contextDir := configs.ContextDir
	if contextDir == "" {
		if wd, err := os.Getwd(); err == nil {
			contextDir = wd
		}
	}

	// Find Dockerfile
	dockerfilePath, err := getDockerfilePath(data.Sdkr, contextDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to find Dockerfile: %v", err)
	}

	// Prepare build args
	buildArgs := make(map[string]string)
	for _, arg := range configs.BuildArgs {
		if parts := strings.SplitN(arg, "=", 2); len(parts) == 2 {
			buildArgs[parts[0]] = parts[1]
		}
	}

	buildOpts := docker.BuildOptions{
		ContextDir:     contextDir,
		DockerfilePath: dockerfilePath,
		NoCache:        configs.NoCache,
		BuildArgs:      buildArgs,
		Target:         configs.Target,
		Platform:       configs.Platform,
		Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
	}

	if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
		return "", "", fmt.Errorf("ECR build failed: %v", err)
	}

	accountID, region, repo, tag, parseErr := configs.ParseEcrImageRef(data.Sdkr.ImageName)
	if parseErr != nil {
		return "", "", parseErr
	}

	fullImage := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s", accountID, region, repo, tag)
	if err := docker.PushImageToECR(fullImage, region, repo); err != nil {
		return "", "", err
	}

	pterm.Success.Printf("✅ Successfully pushed to ECR: %s\n", fullImage)

	if configs.DeleteAfterPush {
		_ = docker.RemoveImage(fullImage)
		pterm.Info.Printf("🧹 Deleted local image: %s\n", fullImage)
	}

	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", accountID, region, repo), tag, nil
}

func handleDockerHubPush(data *configs.Config) (string, string, error) {
	pterm.Info.Println("Pushing image to Docker Hub...")

	username := os.Getenv("DOCKER_USERNAME")
	password := os.Getenv("DOCKER_PASSWORD")

	if username == "" && password == "" {
		envVars := map[string]string{
			"DOCKER_USERNAME": data.Sdkr.DockerUsername,
			"DOCKER_PASSWORD": data.Sdkr.DockerPassword,
		}
		if err := configs.ExportEnvironmentVariables(envVars); err != nil {
			return "", "", err
		}
	}

	localImageName, localTag, err := configs.ParseImage(data.Sdkr.ImageName)
	if err != nil {
		return "", "", fmt.Errorf("invalid image format: %v", err)
	}
	if localTag == "" {
		localTag = "latest"
	}

	fullImageName := fmt.Sprintf("%s:%s", localImageName, localTag)

	// Build args
	buildArgs := make(map[string]string)
	for _, arg := range configs.BuildArgs {
		if parts := strings.SplitN(arg, "=", 2); len(parts) == 2 {
			buildArgs[parts[0]] = parts[1]
		}
	}

	contextDir := configs.ContextDir
	if contextDir == "" {
		if wd, err := os.Getwd(); err == nil {
			contextDir = wd
		}
	}
	dockerfilePath := configs.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = filepath.Join(contextDir, "Dockerfile")
	}

	buildOpts := docker.BuildOptions{
		ContextDir:     contextDir,
		DockerfilePath: dockerfilePath,
		NoCache:        configs.NoCache,
		BuildArgs:      buildArgs,
		Target:         configs.Target,
		Platform:       configs.Platform,
		Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
	}

	if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
		return "", "", err
	}

	pterm.Info.Printf("🚀 Pushing image %s...\n", fullImageName)
	pushOpts := docker.PushOptions{ImageName: fullImageName, Timeout: 1000 * time.Second}

	if err := docker.PushImage(pushOpts); err != nil {
		return "", "", err
	}
	pterm.Success.Printf("✅ Successfully pushed to Docker Hub: %s\n", fullImageName)

	if configs.DeleteAfterPush {
		_ = docker.RemoveImage(fullImageName)
		pterm.Info.Printf("🧹 Deleted local image: %s\n", fullImageName)
	}

	return localImageName, localTag, nil
}

func handleGHCRPush(data *configs.Config) (string, string, error) {
	pterm.Info.Println("Pushing image to GitHub Container Registry (GHCR)...")

	imageName := data.Sdkr.ImageName
	if !strings.HasPrefix(imageName, "ghcr.io/") {
		return "", "", errors.New("GHCR image must start with 'ghcr.io/'")
	}

	if os.Getenv("USERNAME_GITHUB") == "" && os.Getenv("TOKEN_GITHUB") == "" {
		envVars := map[string]string{
			"USERNAME_GITHUB": data.Sdkr.GithubUsername,
			"TOKEN_GITHUB":    data.Sdkr.GithubToken,
		}
		if err := configs.ExportEnvironmentVariables(envVars); err != nil {
			return "", "", err
		}
	}

	localImageName, localTag, err := configs.ParseImage(imageName)
	if err != nil {
		return "", "", fmt.Errorf("invalid image format: %v", err)
	}
	if localTag == "" {
		localTag = "latest"
	}

	fullImage := fmt.Sprintf("%s:%s", localImageName, localTag)

	// Build args
	buildArgs := make(map[string]string)
	for _, arg := range configs.BuildArgs {
		if parts := strings.SplitN(arg, "=", 2); len(parts) == 2 {
			buildArgs[parts[0]] = parts[1]
		}
	}

	contextDir := configs.ContextDir
	if contextDir == "" {
		if wd, err := os.Getwd(); err == nil {
			contextDir = wd
		}
	}
	dockerfilePath := configs.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = filepath.Join(contextDir, "Dockerfile")
	}

	buildOpts := docker.BuildOptions{
		ContextDir:     contextDir,
		DockerfilePath: dockerfilePath,
		NoCache:        configs.NoCache,
		BuildArgs:      buildArgs,
		Target:         configs.Target,
		Platform:       configs.Platform,
		Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
	}

	if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
		return "", "", err
	}

	pterm.Info.Printf("🚀 Pushing image %s to GHCR...\n", fullImage)
	if err := docker.PushToGHCR(docker.PushOptions{
		ImageName: fullImage,
		Timeout:   1000 * time.Second,
	}); err != nil {
		return "", "", err
	}

	pterm.Success.Printf("✅ Successfully pushed to GHCR: %s\n", fullImage)

	if configs.DeleteAfterPush {
		_ = docker.RemoveImage(fullImage)
		pterm.Info.Printf("🧹 Deleted local image: %s\n", fullImage)
	}

	return localImageName, localTag, nil
}

func handleHelmDeploy(data *configs.Config, imageRepo, imageTag string) error {
	pterm.Info.Println("Starting Helm deployment...")

	releaseName := data.Selm.ReleaseName
	if releaseName == "" {
		releaseName = filepath.Base(data.Selm.ChartName)
	}

	chartPath := data.Selm.ChartName
	if releaseName == "" || chartPath == "" {
		return errors.New("release name or chart path missing in config")
	}

	namespace := data.Selm.Namespace
	if namespace == "" {
		namespace = "default"
	}

	valuesFilePath, err := getValuesFilePath(data.Selm, chartPath)
	if err != nil {
		return err
	}

	if imageRepo != "" && imageTag != "" {
		if err := updateValuesYamlFile(valuesFilePath, imageRepo, imageTag); err != nil {
			return fmt.Errorf("failed to update values.yaml: %v", err)
		}
		pterm.Success.Println("✅ Updated values.yaml with new image details")
	}

	timeoutDuration := time.Duration(configs.Timeout) * time.Second

	exists, err := helm.HelmReleaseExists(releaseName, namespace, configs.Debug)
	if err != nil {
		return err
	}

	if !exists {
		pterm.Info.Printf("Installing Helm release %s...\n", releaseName)
		return helm.HelmInstall(
			releaseName,
			chartPath,
			namespace,
			configs.File,
			timeoutDuration,
			configs.Atomic,
			configs.Debug,
			configs.Set,
			configs.SetLiteral,
			"",
			"",
			true,
		)
	}

	pterm.Info.Printf("Upgrading Helm release %s...\n", releaseName)
	return helm.HelmUpgrade(
		releaseName,
		chartPath,
		namespace,
		configs.Set,
		configs.File,
		configs.SetLiteral,
		true,
		configs.Atomic,
		timeoutDuration,
		configs.Debug,
		"",
		"",
		true,
	)
}

// Helper function to update values.yaml file directly
func updateValuesYamlFile(valuesFilePath, imageRepo, imageTag string) error {
	// Read the existing values.yaml file
	content, err := os.ReadFile(valuesFilePath)
	if err != nil {
		return fmt.Errorf("failed to read values.yaml file: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var updatedLines []string

	imageRepoUpdated := false
	imageTagUpdated := false
	inImageSection := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if we're in the image section
		if trimmedLine == "image:" {
			inImageSection = true
			updatedLines = append(updatedLines, line)
			continue
		}

		// If we're in image section, look for repository and tag
		if inImageSection {
			if strings.HasPrefix(trimmedLine, "repository:") {
				if imageRepo != "" {
					updatedLines = append(updatedLines, fmt.Sprintf("  repository: %s", imageRepo))
					imageRepoUpdated = true
					continue
				}
			}

			if strings.HasPrefix(trimmedLine, "tag:") {
				if imageTag != "" {
					if imageTag == "" {
						updatedLines = append(updatedLines, "  tag: \"\"")
					} else {
						updatedLines = append(updatedLines, fmt.Sprintf("  tag: \"%s\"", imageTag))
					}
					imageTagUpdated = true
					continue
				}
			}

			// Check if we're leaving the image section
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, " ") && !strings.HasPrefix(trimmedLine, "\t") {
				inImageSection = false
			}
		}

		updatedLines = append(updatedLines, line)

		// If this is the last line and we need to add image section, add it
		if i == len(lines)-1 && (!imageRepoUpdated || !imageTagUpdated) {
			updatedLines = append(updatedLines, "")
			updatedLines = append(updatedLines, "image:")
			if !imageRepoUpdated && imageRepo != "" {
				updatedLines = append(updatedLines, fmt.Sprintf("  repository: %s", imageRepo))
			}
			if !imageTagUpdated && imageTag != "" {
				if imageTag == "" {
					updatedLines = append(updatedLines, "  tag: \"\"")
				} else {
					updatedLines = append(updatedLines, fmt.Sprintf("  tag: \"%s\"", imageTag))
				}
			}
		}
	}

	// Write the updated content back to the file
	updatedContent := strings.Join(updatedLines, "\n")
	err = os.WriteFile(valuesFilePath, []byte(updatedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write values.yaml file: %v", err)
	}

	return nil
}

// Helper function to get values file path
func getValuesFilePath(selmConfig configs.SelmConfig, chartPath string) (string, error) {
	// If fileName is provided in config, use it
	if selmConfig.FileName != "" {
		// Check if it's an absolute path
		if filepath.IsAbs(selmConfig.FileName) {
			if _, err := os.Stat(selmConfig.FileName); err == nil {
				return selmConfig.FileName, nil
			}
		}

		// If it's relative, check if it exists relative to current directory
		if _, err := os.Stat(selmConfig.FileName); err == nil {
			return selmConfig.FileName, nil
		}

		// If not found, try relative to chart path
		chartDir := filepath.Dir(chartPath)
		potentialPath := filepath.Join(chartDir, selmConfig.FileName)
		if _, err := os.Stat(potentialPath); err == nil {
			return potentialPath, nil
		}

		// Try the path as is (might be relative to current working directory)
		return selmConfig.FileName, nil
	}

	// Default values.yaml in chart directory
	chartDir := filepath.Dir(chartPath)
	defaultValuesPath := filepath.Join(chartDir, "values.yaml")
	if _, err := os.Stat(defaultValuesPath); err == nil {
		return defaultValuesPath, nil
	}

	// If chart is a directory, look for values.yaml inside it
	if info, err := os.Stat(chartPath); err == nil && info.IsDir() {
		defaultValuesPath := filepath.Join(chartPath, "values.yaml")
		if _, err := os.Stat(defaultValuesPath); err == nil {
			return defaultValuesPath, nil
		}
	}

	return "", fmt.Errorf("could not find values.yaml file. Please specify fileName in config")
}

// Helper function to get Dockerfile path
func getDockerfilePath(sdkrConfig configs.SdkrConfig, contextDir string) (string, error) {
	// If Dockerfile is provided in config, use it
	if sdkrConfig.Dockerfile != "" {
		// Check if it's an absolute path
		if filepath.IsAbs(sdkrConfig.Dockerfile) {
			if _, err := os.Stat(sdkrConfig.Dockerfile); err == nil {
				return sdkrConfig.Dockerfile, nil
			}
		}

		// If it's relative, check if it exists relative to current directory
		if _, err := os.Stat(sdkrConfig.Dockerfile); err == nil {
			return sdkrConfig.Dockerfile, nil
		}

		// If not found, try relative to context directory
		potentialPath := filepath.Join(contextDir, sdkrConfig.Dockerfile)
		if _, err := os.Stat(potentialPath); err == nil {
			return potentialPath, nil
		}

		// Try the path as is (might be relative to current working directory)
		return sdkrConfig.Dockerfile, nil
	}

	// Default Dockerfile in context directory
	defaultDockerfile := filepath.Join(contextDir, "Dockerfile")
	if _, err := os.Stat(defaultDockerfile); err == nil {
		return defaultDockerfile, nil
	}

	return "", fmt.Errorf("could not find Dockerfile. Please specify dockerfile in config or ensure Dockerfile exists in context directory")
}
