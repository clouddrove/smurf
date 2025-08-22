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

func Repo_Add(args []string, username, password, certFile, keyFile, caFile, helmConfigDir string) error {
	repoName := args[0]
	repoURL := args[1]

	pterm.Info.Printfln("Adding repo %s...", repoName)

	// Use Helm's default settings
	settings := helmCLI.New()

	// Configure the repository config and cache paths to be compatible with Helm CLI
	if helmConfigDir != "" {
		// Use custom config directory if specified
		settings.RepositoryConfig = filepath.Join(helmConfigDir, "repositories.yaml")
		settings.RepositoryCache = filepath.Join(helmConfigDir, "cache")
	} else {
		// Use the same logic as Helm CLI for default config location
		// Check HELM_HOME environment variable first
		if helmHome := os.Getenv("HELM_HOME"); helmHome != "" {
			settings.RepositoryConfig = filepath.Join(helmHome, "repositories.yaml")
			settings.RepositoryCache = filepath.Join(helmHome, "cache")
		} else {
			// Use default XDG config location (same as Helm CLI)
			configDir := helmpath.ConfigPath()
			settings.RepositoryConfig = filepath.Join(configDir, "repositories.yaml")
			settings.RepositoryCache = filepath.Join(configDir, "cache")
		}
	}

	// Create repository config directory if it doesn't exist
	configDir := filepath.Dir(settings.RepositoryConfig)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create repository config directory: %v", err)
		return fmt.Errorf("failed to create repository config directory: %v", err)
	}

	// Create cache directory if it doesn't exist
	cacheDir := settings.RepositoryCache
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create repository cache directory: %v", err)
		return fmt.Errorf("failed to create repository cache directory: %v", err)
	}

	// Initialize repository config
	var f *repo.File
	if _, err := os.Stat(settings.RepositoryConfig); os.IsNotExist(err) {
		f = repo.NewFile()
	} else {
		f, err = repo.LoadFile(settings.RepositoryConfig)
		if err != nil {
			pterm.Error.Printfln("✗ Failed to load repository file: %v", err)
			return fmt.Errorf("failed to load repository file: %v", err)
		}
	}

	// Check if repository already exists and handle it like Helm CLI
	if existingEntry := f.Get(repoName); existingEntry != nil {
		// Check if the URL is the same
		if existingEntry.URL == repoURL {
			pterm.Info.Printfln("✓ %q already exists with the same configuration", repoName)
			return nil
		} else {
			// URL is different, show error
			pterm.Error.Printfln("✗ repository name (%s) already exists with a different URL", repoName)
			pterm.Println("  Existing URL:", existingEntry.URL)
			pterm.Println("  New URL:     ", repoURL)
			pterm.Info.Println("  If you want to add a repository with a different URL, use a different name")
			return fmt.Errorf("repository name (%s) already exists", repoName)
		}
	}

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
	r, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		pterm.Error.Printfln("✗ Failed to create chart repository: %v", err)
		return fmt.Errorf("failed to create chart repository: %v", err)
	}

	pterm.Info.Println("Downloading repository index...")

	// Download repository index
	start := time.Now()
	if _, err := r.DownloadIndexFile(); err != nil {
		pterm.Error.Printfln("✗ Looks like %q is not a valid chart repository or the URL cannot be reached: %v", repoURL, err)
		return fmt.Errorf("looks like %q is not a valid chart repository or the URL cannot be reached: %v", repoURL, err)
	}

	elapsed := time.Since(start).Truncate(time.Millisecond)
	f.Update(entry)

	if err := f.WriteFile(settings.RepositoryConfig, 0644); err != nil {
		pterm.Error.Printfln("✗ Failed to write repository config: %v", err)
		return fmt.Errorf("failed to write repository config: %v", err)
	}

	pterm.Success.Printfln("✓ Successfully added repo %s", repoName)
	pterm.Println("  Repository has been added to:", settings.RepositoryConfig)
	pterm.Println("  Time elapsed:              ", elapsed)

	return nil
}
