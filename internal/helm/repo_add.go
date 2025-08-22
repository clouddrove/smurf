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

	// CRITICAL: Use the EXACT same settings initialization as Helm CLI
	settings := helmCLI.New()

	// OVERRIDE settings to match exactly what Helm CLI uses
	// This is the key fix - we must use the same paths as the helm command
	if helmConfigDir != "" {
		// If custom config dir provided, use it
		settings.RepositoryConfig = filepath.Join(helmConfigDir, "repositories.yaml")
		settings.RepositoryCache = filepath.Join(helmConfigDir, "repository")
	} else {
		// Use the EXACT same default paths as Helm CLI
		settings.RepositoryConfig = helmpath.ConfigPath("repositories.yaml")
		settings.RepositoryCache = helmpath.CachePath("repository")
	}

	pterm.Debug.Printfln("Repository config path: %s", settings.RepositoryConfig)
	pterm.Debug.Printfln("Repository cache path: %s", settings.RepositoryCache)

	// Create directories with exact same permissions as Helm CLI
	if err := os.MkdirAll(filepath.Dir(settings.RepositoryConfig), 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create config directory: %v", err)
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	if err := os.MkdirAll(settings.RepositoryCache, 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create cache directory: %v", err)
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Load existing repositories or create new file
	var repoFile *repo.File
	if _, err := os.Stat(settings.RepositoryConfig); os.IsNotExist(err) {
		repoFile = repo.NewFile()
	} else {
		repoFile, err = repo.LoadFile(settings.RepositoryConfig)
		if err != nil {
			pterm.Error.Printfln("✗ Failed to load repositories file: %v", err)
			return fmt.Errorf("failed to load repositories file: %v", err)
		}
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

	// Create repository entry with EXACT same structure as Helm CLI
	entry := &repo.Entry{
		Name:     repoName,
		URL:      repoURL,
		Username: username,
		Password: password,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}

	// Create chart repository with proper settings
	chartRepo, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		pterm.Error.Printfln("✗ Failed to create chart repository: %v", err)
		return fmt.Errorf("failed to create chart repository: %v", err)
	}

	pterm.Info.Println("Downloading repository index...")

	// Download index file - this creates the cache that helm pull needs
	start := time.Now()
	indexPath, err := chartRepo.DownloadIndexFile()
	if err != nil {
		pterm.Error.Printfln("✗ Failed to download index: %v", err)
		return fmt.Errorf("failed to download index: %v", err)
	}

	// Add entry to repository file
	repoFile.Update(entry)

	// Write the file with same permissions as Helm CLI
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

	// VERIFICATION: Check if native helm can see the repository
	pterm.Info.Println("Verifying Helm CLI compatibility...")

	// Try to load the file again to verify it's properly formatted
	if verifyFile, err := repo.LoadFile(settings.RepositoryConfig); err == nil {
		if verifyFile.Get(repoName) != nil {
			pterm.Success.Println("✓ Repository successfully registered in Helm configuration")
		} else {
			pterm.Warning.Println("⚠ Repository not found in verification - possible configuration issue")
		}
	}

	return nil
}
