package cmd

import (
	"bufio"
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
	Short:        "Deploy builds and pushes Docker image as per smurf.yaml, then optionally runs Helm deploy.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := configs.LoadConfig(configs.FileName)
		if err != nil {
			return err
		}

		if cfg.Sdkr.ImageName == "" {
			return fmt.Errorf("no image name provided in smurf.yaml or CLI argument")
		}

		var imageRepo, imageTag string

		switch {
		case cfg.Sdkr.AwsECR:
			imageRepo, imageTag, err = handleECRPush(cfg)
		case cfg.Sdkr.DockerHub:
			imageRepo, imageTag, err = handleDockerHubPush(cfg)
		case cfg.Sdkr.GHCRRepo:
			imageRepo, imageTag, err = handleGHCRPush(cfg)
		case cfg.Sdkr.GCPRepo:
			imageRepo, imageTag, err = handleGCPPush(cfg)
		default:
			pterm.Warning.Println("No registry selected (awsECR/dockerHub/ghcrRepo/gcpRepo). Skipping image push.")
		}

		if err != nil {
			return err
		}

		if cfg.Selm.HelmDeploy {
			if err := handleHelmDeploy(cfg, imageRepo, imageTag); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() { RootCmd.AddCommand(deployCmd) }

func buildImageWithOpts(imageName, tag string) error {
	opts, err := prepareDockerBuild()
	if err != nil {
		return err
	}
	return docker.Build(imageName, tag, opts, false)
}

func prepareDockerBuild() (docker.BuildOptions, error) {
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

	buildArgs := make(map[string]string)
	for _, arg := range configs.BuildArgs {
		if parts := strings.SplitN(arg, "=", 2); len(parts) == 2 {
			buildArgs[parts[0]] = parts[1]
		}
	}

	return docker.BuildOptions{
		ContextDir:     contextDir,
		DockerfilePath: dockerfilePath,
		NoCache:        configs.NoCache,
		BuildArgs:      buildArgs,
		Target:         configs.Target,
		Platform:       configs.Platform,
		Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
	}, nil
}

func maybeCleanup(image string) {
	if configs.DeleteAfterPush {
		_ = docker.RemoveImage(image, false)
		pterm.Info.Printf("🧹 Deleted local image: %s\n", image)
	}
}

func handleECRPush(cfg *configs.Config) (string, string, error) {
	pterm.Info.Println("📦 Handling AWS ECR push...")

	accountID, region, repo, tag, err := configs.ParseEcrImageRef(cfg.Sdkr.ImageName)
	if err != nil {
		return "", "", err
	}
	if tag == "" {
		tag = "latest"
	}

	localImage := fmt.Sprintf("%s:%s", repo, tag)
	pterm.Info.Printf("🔧 Building local image %s\n", localImage)
	if err := buildImageWithOpts(repo, tag); err != nil {
		return "", "", err
	}

	fullRemote := fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s", accountID, region, repo, tag)
	pterm.Info.Printf("🚀 Pushing to ECR: %s\n", fullRemote)

	if err := docker.PushImageToECR(fullRemote, region, repo, false); err != nil {
		return "", "", err
	}

	pterm.Success.Printf("✅ Successfully pushed to ECR: %s\n", fullRemote)
	maybeCleanup(localImage)

	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s", accountID, region, repo), tag, nil
}

func handleDockerHubPush(cfg *configs.Config) (string, string, error) {
	pterm.Info.Println("📦 Handling DockerHub push...")

	imageName := cfg.Sdkr.ImageName
	repo, tag := imageName, "latest"
	if parts := strings.SplitN(imageName, ":", 2); len(parts) == 2 {
		repo, tag = parts[0], parts[1]
	}

	if os.Getenv("DOCKER_USERNAME") == "" || os.Getenv("DOCKER_PASSWORD") == "" {
		envVars := map[string]string{
			"DOCKER_USERNAME": cfg.Sdkr.DockerUsername,
			"DOCKER_PASSWORD": cfg.Sdkr.DockerPassword,
		}
		if err := configs.ExportEnvironmentVariables(envVars); err != nil {
			return "", "", err
		}
	}

	pterm.Info.Printf("🔧 Building Docker image %s:%s\n", repo, tag)
	if err := buildImageWithOpts(repo, tag); err != nil {
		return "", "", err
	}

	fullImage := fmt.Sprintf("%s:%s", repo, tag)
	pterm.Info.Printf("🚀 Pushing image %s\n", fullImage)

	if err := docker.PushImage(docker.PushOptions{
		ImageName: fullImage,
		Timeout:   1000 * time.Second,
	}, false); err != nil {
		return "", "", err
	}

	pterm.Success.Printf("✅ Successfully pushed to DockerHub: %s\n", fullImage)
	maybeCleanup(fullImage)

	return repo, tag, nil
}

func handleGHCRPush(cfg *configs.Config) (string, string, error) {
	pterm.Info.Println("📦 Handling GHCR push...")

	imageName := cfg.Sdkr.ImageName
	if !strings.HasPrefix(imageName, "ghcr.io/") {
		return "", "", errors.New("GHCR image must start with 'ghcr.io/'")
	}

	repo, tag := imageName, "latest"
	if parts := strings.SplitN(imageName, ":", 2); len(parts) == 2 {
		repo, tag = parts[0], parts[1]
	}

	if os.Getenv("GITHUB_USERNAME") == "" || os.Getenv("GITHUB_TOKEN") == "" {
		envVars := map[string]string{
			"GITHUB_USERNAME": cfg.Sdkr.GithubUsername,
			"GITHUB_TOKEN":    cfg.Sdkr.GithubToken,
		}
		if err := configs.ExportEnvironmentVariables(envVars); err != nil {
			return "", "", err
		}
	}

	pterm.Info.Printf("🔧 Building GHCR image %s:%s\n", repo, tag)
	if err := buildImageWithOpts(repo, tag); err != nil {
		return "", "", err
	}

	fullImage := fmt.Sprintf("%s:%s", repo, tag)
	pterm.Info.Printf("🚀 Pushing %s to GHCR...\n", fullImage)

	if err := docker.PushToGHCR(docker.PushOptions{
		ImageName: fullImage,
		Timeout:   1000 * time.Second,
	}, false); err != nil {
		return "", "", err
	}

	pterm.Success.Printf("✅ Successfully pushed to GHCR: %s\n", fullImage)
	maybeCleanup(fullImage)

	return repo, tag, nil
}

func handleGCPPush(cfg *configs.Config) (string, string, error) {
	pterm.Info.Println("📦 Handling GCP push...")

	imageName := cfg.Sdkr.ImageName
	repo, tag := imageName, "latest"

	// Parse tag
	if parts := strings.SplitN(imageName, ":", 2); len(parts) == 2 {
		repo, tag = parts[0], parts[1]
	}

	// Validate GCP Registry
	if !strings.HasPrefix(repo, "gcr.io/") && !strings.Contains(repo, "-docker.pkg.dev/") {
		return "", "", fmt.Errorf("invalid GCP registry. Must be gcr.io/ or *.pkg.dev")
	}

	// Extract local build image name
	localRepo := repo
	if strings.Contains(repo, "/") {
		parts := strings.Split(repo, "/")
		localRepo = parts[len(parts)-1]
	}

	// Build local image
	localImageRef := fmt.Sprintf("%s:%s", localRepo, tag)
	pterm.Info.Printf("🔧 Building image %s\n", localImageRef)
	if err := buildImageWithOpts(localRepo, tag); err != nil {
		return "", "", err
	}

	// FULL GCP image reference
	fullRemote := fmt.Sprintf("%s:%s", repo, tag)
	pterm.Info.Printf("🔖 Tagging image: %s → %s\n", localImageRef, fullRemote)

	// Tag
	tagOpts := docker.TagOptions{Source: localImageRef, Target: fullRemote}
	if err := docker.TagImage(tagOpts, false); err != nil {
		return "", "", fmt.Errorf("failed to tag image: %w", err)
	}

	// PUSH using GCP-specific function (like ECR does 🎯)
	if err := docker.PushImageToGCR(configs.ProjectID, fullRemote, false); err != nil {
		return "", "", err
	}

	pterm.Success.Printf("✅ Successfully pushed to GCP: %s\n", fullRemote)
	maybeCleanup(localImageRef)

	// Return repository + tag like ECR function does
	return repo, tag, nil
}

func handleHelmDeploy(data *configs.Config, imageRepo, imageTag string) error {
	pterm.Info.Println("Starting Helm deployment...")

	if strings.Contains(imageRepo, ":") {
		parts := strings.SplitN(imageRepo, ":", 2)
		imageRepo = parts[0]
		pterm.Info.Printf("🧹 Cleaned image repo: %s (removed internal tag)\n", imageRepo)
	}

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

	exists, err := helm.HelmReleaseExists(releaseName, namespace, configs.Debug, false)
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
			false,
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
		3,
		false,
		false,
	)
}

// updateValuesYamlFile updates image.repository and image.tag fields in values.yaml safely.
func updateValuesYamlFile(valuesFilePath, imageRepo, imageTag string) error {
	if imageRepo == "" && imageTag == "" {
		pterm.Warning.Println("⚠️ No imageRepo or imageTag provided, skipping values.yaml update.")
		return nil
	}

	pterm.Info.Printf("🔧 Updating values.yaml: %s\n", valuesFilePath)

	file, err := os.Open(valuesFilePath)
	if err != nil {
		return fmt.Errorf("failed to open values.yaml: %v", err)
	}
	defer file.Close()

	var updatedLines []string
	inImageSection := false
	repoUpdated, tagUpdated := false, false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// detect "image:" section
		if strings.HasPrefix(trimmed, "image:") {
			inImageSection = true
			updatedLines = append(updatedLines, line)
			continue
		}

		if inImageSection {
			// repository line
			if strings.HasPrefix(trimmed, "repository:") && imageRepo != "" {
				indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
				line = fmt.Sprintf("%srepository: %s", indent, imageRepo)
				repoUpdated = true
			}

			// tag line
			if strings.HasPrefix(trimmed, "tag:") && imageTag != "" {
				indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
				line = fmt.Sprintf("%stag: \"%s\"", indent, imageTag)
				tagUpdated = true
			}

			// exit image section if next top-level key found
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") &&
				!strings.HasPrefix(trimmed, "repository:") &&
				!strings.HasPrefix(trimmed, "tag:") &&
				!strings.HasPrefix(line, " ") {
				inImageSection = false
			}
		}

		updatedLines = append(updatedLines, line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading values.yaml: %v", err)
	}

	// if repository/tag not found, append new image section
	if !repoUpdated || !tagUpdated {
		updatedLines = append(updatedLines, "")
		updatedLines = append(updatedLines, "image:")
		if !repoUpdated && imageRepo != "" {
			updatedLines = append(updatedLines, fmt.Sprintf("  repository: %s", imageRepo))
		}
		if !tagUpdated && imageTag != "" {
			updatedLines = append(updatedLines, fmt.Sprintf("  tag: \"%s\"", imageTag))
		}
	}

	// write back
	output := strings.Join(updatedLines, "\n")
	if err := os.WriteFile(valuesFilePath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write updated values.yaml: %v", err)
	}

	pterm.Success.Printf("✅ Updated values.yaml successfully:\n  repository: %s\n  tag: %s\n", imageRepo, imageTag)
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
