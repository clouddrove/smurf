package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

// SearchResult represents a chart search result
type SearchResult struct {
	Name        string
	Version     string
	Description string
	AppVersion  string
	Repository  string
}

// HelmSearchAll searches for charts in all repositories
func HelmSearchAll(keyword, version string, debug bool) ([]SearchResult, error) {
	startTime := time.Now()
	if debug {
		pterm.Println("=== HELM SEARCH ALL STARTED ===")
		pterm.Printf("Keyword: %s\n", keyword)
		pterm.Printf("Version filter: %s\n", version)
	}

	// Load all repository indexes
	indexFiles, err := loadAllIndexFiles(debug)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository indexes: %w", err)
	}

	var results []SearchResult

	// Search through all index files
	for repoName, indexFile := range indexFiles {
		if debug {
			pterm.Printf("Searching repository: %s\n", repoName)
		}

		for chartName, chartVersions := range indexFile.Entries {
			for _, chartVersion := range chartVersions {
				// Apply filters
				if matchesSearch(chartName, chartVersion, keyword, version) {
					results = append(results, SearchResult{
						Name:        chartName,
						Version:     chartVersion.Version,
						Description: chartVersion.Description,
						AppVersion:  chartVersion.AppVersion,
						Repository:  repoName,
					})
				}
			}
		}
	}

	if debug {
		duration := time.Since(startTime)
		pterm.Printf("Search completed in %s, found %d results\n", duration, len(results))
	}

	return results, nil
}

// HelmSearchRepo searches for charts in specific repositories
func HelmSearchRepo(keyword, version, repoURL string, debug bool) ([]SearchResult, error) {
	startTime := time.Now()
	if debug {
		pterm.Println("=== HELM SEARCH REPO STARTED ===")
		pterm.Printf("Keyword: %s\n", keyword)
		pterm.Printf("Version filter: %s\n", version)
		pterm.Printf("Repo URL: %s\n", repoURL)
	}

	// Load all repository indexes
	indexFiles, err := loadAllIndexFiles(debug)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository indexes: %w", err)
	}

	var results []SearchResult

	// Search through specific repositories
	for repoName, indexFile := range indexFiles {
		// If repoURL is specified, filter by repository URL
		if repoURL != "" {
			repoEntry, err := getRepositoryByName(repoName)
			if err != nil || repoEntry.URL != repoURL {
				continue
			}
		}

		if debug {
			pterm.Printf("Searching repository: %s\n", repoName)
		}

		for chartName, chartVersions := range indexFile.Entries {
			for _, chartVersion := range chartVersions {
				// Apply filters
				if matchesSearch(chartName, chartVersion, keyword, version) {
					results = append(results, SearchResult{
						Name:        chartName,
						Version:     chartVersion.Version,
						Description: chartVersion.Description,
						AppVersion:  chartVersion.AppVersion,
						Repository:  repoName,
					})
				}
			}
		}
	}

	if debug {
		duration := time.Since(startTime)
		pterm.Printf("Repository search completed in %s, found %d results\n", duration, len(results))
	}

	return results, nil
}

// matchesSearch checks if a chart matches the search criteria
func matchesSearch(chartName string, chartVersion *repo.ChartVersion, keyword, version string) bool {
	// If no keyword provided, match all (unless version filter is applied)
	if keyword == "" && version == "" {
		return true
	}

	// Check version filter
	if version != "" && chartVersion.Version != version {
		return false
	}

	// Check keyword filter
	if keyword != "" {
		keyword = strings.ToLower(keyword)
		nameMatch := strings.Contains(strings.ToLower(chartName), keyword)
		descMatch := strings.Contains(strings.ToLower(chartVersion.Description), keyword)

		if !nameMatch && !descMatch {
			return false
		}
	}

	return true
}

// loadAllIndexFiles loads index files from all repositories
func loadAllIndexFiles(debug bool) (map[string]*repo.IndexFile, error) {
	settings := cli.New()
	repoFile := settings.RepositoryConfig
	cacheDir := settings.RepositoryCache

	if debug {
		pterm.Printf("Repository file: %s\n", repoFile)
		pterm.Printf("Cache directory: %s\n", cacheDir)
	}

	// Load repository configuration
	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no repositories configured. Please add a repository first")
	} else if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	if debug {
		pterm.Printf("Found %d repositories\n", len(f.Repositories))
	}

	indexFiles := make(map[string]*repo.IndexFile)

	// Load index files for each repository
	for _, repoEntry := range f.Repositories {
		indexFilePath := filepath.Join(cacheDir, repoEntry.Name+"-index.yaml")

		if debug {
			pterm.Printf("Loading index for %s: %s\n", repoEntry.Name, indexFilePath)
		}

		indexFile, err := repo.LoadIndexFile(indexFilePath)
		if err != nil {
			if debug {
				pterm.Warning.Printf("Failed to load index for %s: %v\n", repoEntry.Name, err)
			}
			continue
		}

		indexFiles[repoEntry.Name] = indexFile
	}

	if len(indexFiles) == 0 {
		return nil, fmt.Errorf("no repository indexes found. Run 'helm repo update' first")
	}

	return indexFiles, nil
}

// getRepositoryByName returns a repository entry by name
func getRepositoryByName(name string) (*repo.Entry, error) {
	settings := cli.New()
	repoFile := settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return nil, err
	}

	for _, repoEntry := range f.Repositories {
		if repoEntry.Name == name {
			return repoEntry, nil
		}
	}

	return nil, fmt.Errorf("repository %s not found", name)
}

// ListRepositories lists all configured Helm repositories
func ListRepositories(debug bool) ([]*repo.Entry, error) {
	if debug {
		pterm.Println("Listing configured repositories...")
	}

	settings := cli.New()
	repoFile := settings.RepositoryConfig

	if debug {
		pterm.Printf("Repository file: %s\n", repoFile)
	}

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	if debug {
		pterm.Printf("Found %d repositories\n", len(f.Repositories))
		for _, repo := range f.Repositories {
			pterm.Printf("  - %s: %s\n", repo.Name, repo.URL)
		}
	}

	return f.Repositories, nil
}

// AddRepository adds a new Helm repository
func AddRepository(name, url string, debug bool) error {
	if debug {
		pterm.Printf("Adding repository: %s -> %s\n", name, url)
	}

	settings := cli.New()
	repoFile := settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(err) {
		f = &repo.File{}
	} else if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	// Check if repository already exists
	for _, repo := range f.Repositories {
		if repo.Name == name {
			if debug {
				pterm.Printf("Repository %s already exists\n", name)
			}
			return fmt.Errorf("repository %s already exists", name)
		}
		if repo.URL == url {
			if debug {
				pterm.Printf("Repository with URL %s already exists as %s\n", url, repo.Name)
			}
			return fmt.Errorf("repository with URL %s already exists as %s", url, repo.Name)
		}
	}

	// Add new repository
	newRepo := &repo.Entry{
		Name: name,
		URL:  url,
	}

	f.Update(newRepo)

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	// Update repository index
	if debug {
		pterm.Println("Updating repository index...")
	}

	chartRepo, err := repo.NewChartRepository(newRepo, getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return fmt.Errorf("failed to download index: %w", err)
	}

	if debug {
		pterm.Success.Printf("Repository %s added successfully\n", name)
	}

	return nil
}

// RemoveRepository removes a Helm repository
func RemoveRepository(name string, debug bool) error {
	if debug {
		pterm.Printf("Removing repository: %s\n", name)
	}

	settings := cli.New()
	repoFile := settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	// Find and remove repository
	found := false
	var updatedRepos []*repo.Entry
	for _, repo := range f.Repositories {
		if repo.Name == name {
			found = true
			continue
		}
		updatedRepos = append(updatedRepos, repo)
	}

	if !found {
		return fmt.Errorf("repository %s not found", name)
	}

	f.Repositories = updatedRepos

	if err := f.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	if debug {
		pterm.Success.Printf("Repository %s removed successfully\n", name)
	}

	return nil
}

// UpdateRepositories updates all repository indexes
func UpdateRepositories(debug bool) error {
	if debug {
		pterm.Println("Updating all repository indexes...")
	}

	settings := cli.New()
	repoFile := settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	if debug {
		pterm.Printf("Found %d repositories to update\n", len(f.Repositories))
	}

	for _, repoEntry := range f.Repositories {
		if debug {
			pterm.Printf("Updating repository: %s\n", repoEntry.Name)
		}

		chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(settings))
		if err != nil {
			pterm.Warning.Printf("Failed to update repository %s: %v\n", repoEntry.Name, err)
			continue
		}

		if _, err := chartRepo.DownloadIndexFile(); err != nil {
			pterm.Warning.Printf("Failed to download index for %s: %v\n", repoEntry.Name, err)
			continue
		}

		if debug {
			pterm.Printf("Repository %s updated successfully\n", repoEntry.Name)
		}
	}

	if debug {
		pterm.Success.Println("All repositories updated successfully")
	}

	return nil
}
