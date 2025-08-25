package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func Repo_Add(args []string, username, password, certFile, keyFile, caFile, helmConfigDir string) error {
	repoName := args[0]
	repoURL := args[1]

	pterm.Info.Printfln("Adding repo %s...", repoName)

	// Get Helm settings the EXACT way Helm CLI does
	settings := getHelmSettings(helmConfigDir)

	// Debug output
	pterm.Debug.Printfln("Using repository config: %s", settings.RepositoryConfig)
	pterm.Debug.Printfln("Using repository cache: %s", settings.RepositoryCache)

	// Ensure directories exist with correct permissions
	if err := ensureDirectoriesExist(settings); err != nil {
		return err
	}

	// Load or create repository file using Helm's method
	repoFile, err := loadOrCreateRepoFile(settings.RepositoryConfig)
	if err != nil {
		return err
	}

	// Check if repository already exists
	if existing := repoFile.Get(repoName); existing != nil {
		if existing.URL == repoURL {
			pterm.Info.Printfln("✓ %q already exists with the same configuration", repoName)
			return nil
		}
		pterm.Error.Printfln("✗ Repository %s already exists with different URL", repoName)
		pterm.Println("  Existing URL:", existing.URL)
		pterm.Println("  New URL:     ", repoURL)
		return fmt.Errorf("repository %s already exists", repoName)
	}

	// Create and test the repository
	if err := createAndTestRepository(repoFile, repoName, repoURL, username, password, certFile, keyFile, caFile, settings); err != nil {
		return err
	}

	// Verify the repository is accessible to Helm CLI
	return verifyHelmCompatibility(settings.RepositoryConfig, repoName)
}

// getHelmSettings returns settings configured EXACTLY like Helm CLI
func getHelmSettings(helmConfigDir string) *helmCLI.EnvSettings {
	// Use the default Helm settings (this loads environment variables, etc.)
	settings := helmCLI.New()

	if helmConfigDir != "" {
		// Override with custom directory
		settings.RepositoryConfig = filepath.Join(helmConfigDir, "repositories.yaml")
		settings.RepositoryCache = filepath.Join(helmConfigDir, "repository")
	}
	// Otherwise, use Helm's defaults which respect HELM_HOME, XDG_CONFIG_HOME, etc.

	return settings
}

func ensureDirectoriesExist(settings *helmCLI.EnvSettings) error {
	// Ensure config directory exists with proper permissions
	configDir := filepath.Dir(settings.RepositoryConfig)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create config directory %s: %v", configDir, err)
		return fmt.Errorf("failed to create config directory %s: %v", configDir, err)
	}
	pterm.Debug.Printfln("✓ Config directory exists: %s", configDir)

	// Ensure cache directory exists with proper permissions
	if err := os.MkdirAll(settings.RepositoryCache, 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create cache directory %s: %v", settings.RepositoryCache, err)
		return fmt.Errorf("failed to create cache directory %s: %v", settings.RepositoryCache, err)
	}
	pterm.Debug.Printfln("✓ Cache directory exists: %s", settings.RepositoryCache)

	return nil
}

func loadOrCreateRepoFile(configPath string) (*repo.File, error) {
	// First, ensure the directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory %s: %v", configDir, err)
	}

	// Try to load existing file
	repoFile, err := repo.LoadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			pterm.Info.Printfln("Creating new repositories file at %s", configPath)
			// Create empty repositories file
			newRepoFile := repo.NewFile()

			// Write empty file to ensure directory structure is valid
			if err := newRepoFile.WriteFile(configPath, 0644); err != nil {
				return nil, fmt.Errorf("failed to create repositories file: %v", err)
			}

			pterm.Success.Printfln("✓ Created new repositories file")
			return newRepoFile, nil
		}
		pterm.Error.Printfln("✗ Failed to load repositories file: %v", err)
		return nil, fmt.Errorf("failed to load repositories file: %v", err)
	}

	pterm.Debug.Printfln("Loaded existing repositories file with %d entries", len(repoFile.Repositories))
	return repoFile, nil
}

func createAndTestRepository(repoFile *repo.File, repoName, repoURL, username, password, certFile, keyFile, caFile string, settings *helmCLI.EnvSettings) error {
	// Create repository entry
	entry := &repo.Entry{
		Name:     repoName,
		URL:      repoURL,
		Username: username,
		Password: password,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	// Create chart repository
	chartRepo, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		pterm.Error.Printfln("✗ Failed to create chart repository: %v", err)
		return fmt.Errorf("failed to create chart repository: %v", err)
	}

	// Download index file with progress indicator
	spinner, _ := pterm.DefaultSpinner.Start("Downloading repository index from " + repoURL)
	start := time.Now()

	indexPath, err := chartRepo.DownloadIndexFile()
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to download index: %v", err))
		return fmt.Errorf("failed to download index: %v", err)
	}

	// Add entry to repository file
	repoFile.Update(entry)

	// CRITICAL: Use Helm's method to write the file
	if err := repoFile.WriteFile(settings.RepositoryConfig, 0644); err != nil {
		spinner.Fail("Failed to write repositories file")
		return fmt.Errorf("failed to write repositories file: %v", err)
	}

	elapsed := time.Since(start).Truncate(time.Millisecond)
	spinner.Success("Repository index downloaded successfully")

	pterm.Success.Printfln("✓ Successfully added repo %s", repoName)
	pterm.Info.Printfln("  Config: %s", settings.RepositoryConfig)
	pterm.Info.Printfln("  Cache:  %s", settings.RepositoryCache)
	pterm.Info.Printfln("  Index:  %s", indexPath)
	pterm.Info.Printfln("  Time:   %v", elapsed)

	return nil
}

func verifyHelmCompatibility(configPath, repoName string) error {
	pterm.Info.Println("Verifying Helm CLI compatibility...")

	// Try to load the file using the same method Helm CLI would use
	repoFile, err := repo.LoadFile(configPath)
	if err != nil {
		pterm.Warning.Printfln("⚠ Could not verify configuration: %v", err)
		return nil
	}

	if repoFile.Get(repoName) == nil {
		pterm.Error.Println("✗ CRITICAL: Repository was not found in configuration file after writing")
		pterm.Info.Println("This indicates a serious compatibility issue with Helm CLI")
		return fmt.Errorf("repository not found in configuration after write")
	}

	pterm.Success.Println("✓ Repository successfully registered and compatible with Helm CLI")

	// Show all repositories
	pterm.Info.Printfln("Available repositories: %v", getRepositoryNames(repoFile))

	return nil
}

// DebugHelmPaths provides diagnostic information about Helm paths
func DebugHelmPaths() {
	pterm.Info.Println("=== Helm Path Debug Information ===")

	// Check what the native Helm CLI would use
	settings := helmCLI.New()
	pterm.Info.Printf("Helm CLI RepositoryConfig: %s\n", settings.RepositoryConfig)
	pterm.Info.Printf("Helm CLI RepositoryCache: %s\n", settings.RepositoryCache)

	// Check environment variables
	pterm.Info.Printf("HELM_HOME: %s\n", os.Getenv("HELM_HOME"))
	pterm.Info.Printf("XDG_CONFIG_HOME: %s\n", os.Getenv("XDG_CONFIG_HOME"))
	pterm.Info.Printf("XDG_CACHE_HOME: %s\n", os.Getenv("XDG_CACHE_HOME"))
	pterm.Info.Printf("HOME: %s\n", os.Getenv("HOME"))

	// Check if config directory exists
	configDir := filepath.Dir(settings.RepositoryConfig)
	if _, err := os.Stat(configDir); err == nil {
		pterm.Success.Printf("Config directory exists: %s\n", configDir)
	} else {
		pterm.Warning.Printf("Config directory does not exist: %s\n", configDir)
		pterm.Info.Printf("Error: %v\n", err)
	}

	// Check if repositories file exists
	if _, err := os.Stat(settings.RepositoryConfig); err == nil {
		pterm.Success.Printf("repositories.yaml exists: %s\n", settings.RepositoryConfig)
		if repoFile, err := repo.LoadFile(settings.RepositoryConfig); err == nil {
			pterm.Info.Printf("Repositories found: %v\n", getRepositoryNames(repoFile))
		} else {
			pterm.Warning.Printf("Failed to load repositories file: %v\n", err)
		}
	} else {
		pterm.Warning.Printf("repositories.yaml does not exist: %s\n", settings.RepositoryConfig)
		pterm.Info.Printf("Error: %v\n", err)
	}

	// Check cache directory
	if _, err := os.Stat(settings.RepositoryCache); err == nil {
		pterm.Success.Printf("Repository cache exists: %s\n", settings.RepositoryCache)
		// List files in cache directory
		if files, err := os.ReadDir(settings.RepositoryCache); err == nil {
			pterm.Info.Printf("Cache files: %d items\n", len(files))
			for _, file := range files {
				pterm.Debug.Printf("  - %s\n", file.Name())
			}
		}
	} else {
		pterm.Warning.Printf("Repository cache does not exist: %s\n", settings.RepositoryCache)
		pterm.Info.Printf("Error: %v\n", err)
	}

	pterm.Info.Println("===================================")
}

func getRepositoryNames(repoFile *repo.File) []string {
	names := make([]string, len(repoFile.Repositories))
	for i, repo := range repoFile.Repositories {
		names[i] = repo.Name
	}
	return names
}
