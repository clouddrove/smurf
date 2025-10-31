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

// Helper function to update image values in setValues
func updateImageValues(setValues *[]string, imageRepo, imageTag string) {
	if imageRepo != "" {
		*setValues = append(*setValues, fmt.Sprintf("image.repository=%s", imageRepo))
	}
	if imageTag != "" {
		*setValues = append(*setValues, fmt.Sprintf("image.tag=%s", imageTag))
	}
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

var deployCmd = &cobra.Command{
	Use:          "deploy",
	Short:        "Deploy is used to perform operation provided smurf.yml file configuration.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := configs.LoadConfig(configs.FileName)
		if err != nil {
			return err
		}

		imageName := data.Sdkr.ImageName
		if imageName == "" {
			return fmt.Errorf("%v", "No image name provided. Please provide an image name as an argument or in the config.")
		}
		tag := data.Sdkr.TargetImageTag
		if tag == "" {
			tag = "latest"
		}
		var imageRepo, imageTag string

		if data.Sdkr.AwsECR {
			localImageName, localTag, parseErr := configs.ParseImage(imageName)
			if parseErr != nil {
				return fmt.Errorf("invalid image format: %v", parseErr)
			}

			if localTag == "" {
				localTag = "latest"
			}

			// Determine context directory
			contextDir := configs.ContextDir
			if contextDir == "" {
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current working directory: %w", err)
				}
				contextDir = wd
			}

			// Get Dockerfile path dynamically
			dockerfilePath, err := getDockerfilePath(data.Sdkr, contextDir)
			if err != nil {
				return fmt.Errorf("failed to find Dockerfile: %v", err)
			}

			pterm.Info.Printf("Using Dockerfile: %s\n", dockerfilePath)
			pterm.Info.Printf("Using build context: %s\n", contextDir)

			fullEcrImage := fmt.Sprintf(
				"%s.dkr.ecr.%s.amazonaws.com/%s:%s",
				localImageName,
				configs.Region,
				configs.Repository,
				localTag,
			)

			buildArgsMap := make(map[string]string)
			for _, arg := range configs.BuildArgs {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					buildArgsMap[parts[0]] = parts[1]
				}
			}

			buildOpts := docker.BuildOptions{
				ContextDir:     contextDir,
				DockerfilePath: dockerfilePath,
				NoCache:        configs.NoCache,
				BuildArgs:      buildArgsMap,
				Target:         configs.Target,
				Platform:       configs.Platform,
				Timeout:        time.Duration(configs.BuildTimeout) * time.Second,
			}

			if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
				return fmt.Errorf("build failed: %v", err)
			}

			pushImage := imageName
			if pushImage == "" {
				pushImage = fullEcrImage
			}

			accountID, ecrRegionName, ecrRepositoryName, ecrImageTag, parseErr := configs.ParseEcrImageRef(imageName)
			if parseErr != nil {
				return parseErr
			}

			if accountID == "" || ecrRegionName == "" || ecrRepositoryName == "" || ecrImageTag == "" {
				return errors.New("invalid image reference: missing account ID, region, or repository name")
			}
			imageName = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s:%s",
				accountID, ecrRegionName, ecrRepositoryName, ecrImageTag,
			)

			// Store image repo and tag for Helm deployment
			imageRepo = fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com/%s",
				accountID, ecrRegionName, ecrRepositoryName)
			imageTag = ecrImageTag

			pterm.Info.Printf("Pushing image %s to ECR...\n", pushImage)
			if err := docker.PushImageToECR(imageName, ecrRegionName, ecrRepositoryName); err != nil {
				return err
			}
			pterm.Success.Println("Push to ECR completed successfully.")

			if configs.DeleteAfterPush {
				pterm.Info.Printf("Deleting local image %s...\n", fullEcrImage)
				if err := docker.RemoveImage(fullEcrImage); err != nil {
					return err
				}
				pterm.Success.Println("Successfully deleted local image:", fullEcrImage)
			}
		}

		if data.Sdkr.DockerHub {
			if os.Getenv("DOCKER_USERNAME") == "" && os.Getenv("DOCKER_PASSWORD") == "" {
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

			localImageName, localTag, parseErr := configs.ParseImage(imageName)
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

			imageRepo = localImageName
			imageTag = localTag
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
			if err := docker.Build(localImageName, localTag, buildOpts); err != nil {
				return err
			}
			pterm.Success.Println("Build completed successfully.")
			pterm.Info.Printf("Pushing image %s...\n", fullImageName)
			pushOpts := docker.PushOptions{
				ImageName: fullImageName,
				Timeout:   1000000000000,
			}
			if err := docker.PushImage(pushOpts); err != nil {
				pterm.Error.Println("Push failed:", err)
				return err
			}

			if configs.DeleteAfterPush {
				pterm.Info.Printf("Deleting local image %s...\n", fullImageName)
				if err := docker.RemoveImage(fullImageName); err != nil {
					pterm.Error.Println("Failed to delete local image:", err)
					return err
				}
				pterm.Success.Println("Successfully deleted local image:", fullImageName)
			}

		}

		if data.Sdkr.GHCRRepo {
			// Validate that image reference includes GHCR registry
			if !strings.HasPrefix(imageName, "ghcr.io/") {
				pterm.Error.Printfln("GHCR image must start with 'ghcr.io/', got: %s", imageName)
				pterm.Info.Println("GHCR images must be in the format: ghcr.io/OWNER/IMAGE_NAME:TAG")
				return errors.New("invalid GHCR image format")
			}

			// Check for GitHub credentials
			if os.Getenv("GITHUB_USERNAME") == "" && os.Getenv("GITHUB_TOKEN") == "" {
				data, err := configs.LoadConfig(configs.FileName)
				if err != nil {
					return err
				}

				envVars := map[string]string{
					"GITHUB_USERNAME": data.Sdkr.GithubUsername,
					"GITHUB_TOKEN":    data.Sdkr.GithubToken,
				}
				if err := configs.ExportEnvironmentVariables(envVars); err != nil {
					return err
				}
			}

			if os.Getenv("GITHUB_USERNAME") == "" || os.Getenv("GITHUB_TOKEN") == "" {
				pterm.Error.Println("GitHub Container Registry credentials are required")
				pterm.Info.Println("You can set them via environment variables:")
				pterm.Info.Println("  export GITHUB_USERNAME=\"your-username\"")
				pterm.Info.Println("  export GITHUB_TOKEN=\"your-github-personal-access-token\"")
				pterm.Info.Println("The token must have 'write:packages' scope")
				pterm.Info.Println("Or set them in your config file as github_username and github_token")
				return errors.New("missing required GitHub Container Registry credentials")
			}

			// Parse image reference
			localImageName, localTag, parseErr := configs.ParseImage(imageName)
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
			imageRepo = localImageName
			imageTag = localTag

			// Login to GHCR
			pterm.Info.Println("Logging in to GitHub Container Registry...")
			loginOpts := docker.LoginOptions{
				Registry: "ghcr.io",
				Username: os.Getenv("GITHUB_USERNAME"),
				Password: os.Getenv("GITHUB_TOKEN"),
			}
			if err := docker.Login(loginOpts); err != nil {
				pterm.Error.Println("GHCR login failed:", err)
				return fmt.Errorf("GHCR login failed: %v", err)
			}
			pterm.Success.Println("Successfully logged in to GitHub Container Registry")

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
		}

		if data.Selm.HelmDeploy {
			if configs.Debug {
				pterm.EnableDebugMessages()
				pterm.Println("=== DEBUG MODE ENABLED ===")
			}

			releaseName := data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			chartPath := data.Selm.ChartName

			if releaseName == "" || chartPath == "" {
				return errors.New("RELEASE and CHART must be provided either as arguments or in the config")
			}

			namespace := data.Selm.Namespace
			if namespace == "" {
				namespace = "default"
			}

			// Get values file path
			valuesFilePath, err := getValuesFilePath(data.Selm, chartPath)
			if err != nil {
				return err
			}

			pterm.Info.Printf("Using values file: %s\n", valuesFilePath)

			// Update values.yaml file with new image details
			if imageRepo != "" || imageTag != "" {
				pterm.Info.Printf("Updating values.yaml with image - Repository: %s, Tag: %s\n", imageRepo, imageTag)
				if err := updateValuesYamlFile(valuesFilePath, imageRepo, imageTag); err != nil {
					return fmt.Errorf("failed to update values.yaml: %v", err)
				}
				pterm.Success.Println("Successfully updated values.yaml file")
			}

			timeoutDuration := time.Duration(configs.Timeout) * time.Second

			createNamespace := true
			installIfNotPresent := true
			wait := true
			RepoURL := ""
			Version := ""

			// Prepare set values for Helm (as backup)
			setValues := configs.Set
			if setValues == nil {
				setValues = []string{}
			}

			// Also add image values to setValues as backup
			if imageRepo != "" || imageTag != "" {
				updateImageValues(&setValues, imageRepo, imageTag)
			}

			if configs.Debug {
				pterm.Printf("Configuration\n")
				pterm.Printf("  - Release: %s\n", releaseName)
				pterm.Printf("  - Chart: %s\n", chartPath)
				pterm.Printf("  - Namespace: %s\n", namespace)
				pterm.Printf("  - Values File: %s\n", valuesFilePath)
				pterm.Printf("  - Timeout: %v\n", timeoutDuration)
				pterm.Printf("  - Atomic: %t\n", configs.Atomic)
				pterm.Printf("  - Create Namespace: %t\n", createNamespace)
				pterm.Printf("  - Install if not present: %t\n", installIfNotPresent)
				pterm.Printf("  - Wait: %t\n", wait)
				pterm.Printf("  - Set values: %v\n", setValues)
				pterm.Printf("  - Values files: %v\n", configs.File)
				pterm.Printf("  - Set literal: %v\n", configs.SetLiteral)
				pterm.Printf("  - Repo URL: %s\n", RepoURL)
				pterm.Printf("  - Version: %s\n", Version)
				if imageName != "" {
					pterm.Printf("  - ECR Image: %s\n", imageName)
				}
			}

			// Check if release exists
			exists, err := helm.HelmReleaseExists(releaseName, namespace, configs.Debug)
			if err != nil {
				return err
			}

			if !exists {
				if installIfNotPresent {
					if configs.Debug {
						pterm.Println("Release not found, installing...")
					}
					if err := helm.HelmInstall(releaseName, chartPath, namespace, configs.File, timeoutDuration, configs.Atomic, configs.Debug, setValues, configs.SetLiteral, RepoURL, Version, wait); err != nil {
						return err
					}
					if configs.Debug {
						pterm.Println("Installation completed successfully")
					}
					pterm.Success.Println("Helm chart installed successfully.")
					return nil
				} else {
					return fmt.Errorf("release %s not found in namespace %s. Use --install flag to install it", releaseName, namespace)
				}
			}

			if configs.Debug {
				pterm.Println("Release exists, proceeding with upgrade")
			}

			if configs.Debug {
				pterm.Println("Starting Helm upgrade...")
			}

			err = helm.HelmUpgrade(
				releaseName,
				chartPath,
				namespace,
				setValues,
				configs.File,
				configs.SetLiteral,
				createNamespace,
				configs.Atomic,
				timeoutDuration,
				configs.Debug,
				RepoURL,
				Version,
				wait,
			)
			if err != nil {
				return fmt.Errorf("helm upgrade failed: %v", err)
			}

			pterm.Success.Println("Helm deployment completed successfully.")
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(deployCmd)
}
