package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
)

func Repo_Add(args []string,
	username, password, certFile, keyFile, caFile, helmConfigDir string,
	useAI bool,
) error {
	repoName := args[0]
	repoURL := args[1]

	pterm.Info.Printfln("Adding repo %s...", repoName)

	// CRITICAL: Get settings the SAME way Helm CLI does
	settings := getHelmSettings(helmConfigDir)

	pterm.Debug.Printfln("Using repository config: %s", settings.RepositoryConfig)
	pterm.Debug.Printfln("Using repository cache: %s", settings.RepositoryCache)

	// Ensure directories exist with correct permissions
	if err := os.MkdirAll(filepath.Dir(settings.RepositoryConfig), 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create config directory: %v", err)
		aiExplainError(useAI, err.Error())
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	if err := os.MkdirAll(settings.RepositoryCache, 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create cache directory: %v", err)
		aiExplainError(useAI, err.Error())
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Load or create repository file
	repoFile, err := loadOrCreateRepoFile(settings.RepositoryConfig)
	if err != nil {
		aiExplainError(useAI, err.Error())
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
		errMsg := fmt.Sprintf("repository %s already exists", repoName)
		aiExplainError(useAI, errMsg)
		return fmt.Errorf("%v", errMsg)
	}

	// Create and test the repository
	if err := createAndTestRepository(repoFile, repoName, repoURL, username, password, certFile, keyFile, caFile, settings); err != nil {
		aiExplainError(useAI, err.Error())
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
	} else {
		// Use the EXACT same defaults as Helm CLI
		settings.RepositoryConfig = helmpath.ConfigPath("repositories.yaml")
		settings.RepositoryCache = helmpath.CachePath("repository")
	}

	return settings
}

func loadOrCreateRepoFile(configPath string) (*repo.File, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		pterm.Debug.Println("Creating new repositories file")
		return repo.NewFile(), nil
	}

	repoFile, err := repo.LoadFile(configPath)
	if err != nil {
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

	pterm.Info.Println("Downloading repository index...")

	// Download index file
	start := time.Now()
	indexPath, err := chartRepo.DownloadIndexFile()
	if err != nil {
		pterm.Error.Printfln("✗ Failed to download index: %v", err)
		return fmt.Errorf("failed to download index: %v", err)
	}

	// Add entry to repository file
	repoFile.Update(entry)

	// Write the file
	if err := repoFile.WriteFile(settings.RepositoryConfig, 0644); err != nil {
		pterm.Error.Printfln("✗ Failed to write repositories file: %v", err)
		return fmt.Errorf("failed to write repositories file: %v", err)
	}

	elapsed := time.Since(start).Truncate(time.Millisecond)

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

	// Show what's actually in the file
	pterm.Debug.Printfln("Current repositories in file: %v", getRepositoryNames(repoFile))

	return nil
}

func getRepositoryNames(repoFile *repo.File) []string {
	names := make([]string, len(repoFile.Repositories))
	for i, repo := range repoFile.Repositories {
		names[i] = repo.Name
	}
	return names
}

// Add this diagnostic function
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

	// Check if files exist
	if _, err := os.Stat(settings.RepositoryConfig); err == nil {
		pterm.Success.Printf("repositories.yaml exists: %s\n", settings.RepositoryConfig)
		if repoFile, err := repo.LoadFile(settings.RepositoryConfig); err == nil {
			pterm.Info.Printf("Repositories found: %v\n", getRepositoryNames(repoFile))
		}
	} else {
		pterm.Warning.Printf("repositories.yaml does not exist: %s\n", settings.RepositoryConfig)
	}

	pterm.Info.Println("===================================")
}
