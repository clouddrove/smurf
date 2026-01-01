package helm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
)

func Repo_Update(args []string, helmConfigDir string, useAI bool) error {
	// Use Helm's default settings
	settings := helmCLI.New()

	// Configure the repository config path to be compatible with Helm CLI
	if helmConfigDir != "" {
		settings.RepositoryConfig = filepath.Join(helmConfigDir, "repositories.yaml")
		settings.RepositoryCache = filepath.Join(helmConfigDir, "cache")
	} else {
		// Use the exact same paths as Helm CLI
		settings.RepositoryConfig = helmpath.ConfigPath("repositories.yaml")
		settings.RepositoryCache = helmpath.CachePath("repository")
	}

	pterm.Info.Println("Hang tight while we grab the latest from your chart repositories...")

	// Load repository file
	f, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		if os.IsNotExist(err) {
			pterm.Error.Println("✗ No repositories found. You must add one first")
			return errors.New("no repositories found")
		}
		pterm.Error.Printfln("✗ Failed to load repository config: %v", err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to load repository config: %v", err)
	}

	var repos []*repo.ChartRepository
	updateAll := len(args) == 0

	// Filter repositories
	for _, cfg := range f.Repositories {
		if !updateAll && !containsRepo(args, cfg.Name) {
			continue
		}

		r, err := repo.NewChartRepository(cfg, getter.All(settings))
		if err != nil {
			pterm.Warning.Printfln("⚠ Failed to create chart repository for %s: %v", cfg.Name, err)
			ai.AIExplainError(useAI, err.Error())
			continue
		}
		repos = append(repos, r)
	}

	// Validate repositories
	if len(repos) == 0 {
		if updateAll {
			pterm.Error.Println("✗ No repositories found. You must add one first.")
			return errors.New("no repositories found")
		}
		pterm.Error.Println("✗ No repositories found matching the provided names")
		return errors.New("no repositories found matching the provided names")
	}

	// Prepare progress tracking
	totalRepos := len(repos)
	successRepos := make([]string, 0, totalRepos)
	failedRepos := make([]string, 0, totalRepos)

	pterm.Info.Printfln("Updating %d chart repositor%s...", totalRepos, pluralize(totalRepos, "y", "ies"))

	start := time.Now()

	// Update repositories
	for _, r := range repos {
		repoName := r.Config.Name
		pterm.Info.Printfln("Fetching latest index for %s...", repoName)

		if _, err := r.DownloadIndexFile(); err != nil {
			failedRepos = append(failedRepos, r.Config.Name)
			pterm.Warning.Printfln("⚠ Failed to update repository %s: %v", r.Config.Name, err)
			continue
		}

		successRepos = append(successRepos, r.Config.Name)
		pterm.Success.Printfln("✓ Successfully got an update from the %s chart repository", r.Config.Name)
	}

	// Final summary
	elapsed := time.Since(start).Truncate(time.Millisecond)

	// Print detailed summary
	pterm.Println()
	pterm.Info.Println("Update Summary:")
	pterm.Info.Printf("  Total Repositories: %d\n", totalRepos)
	pterm.Info.Printf("  Successfully Updated: %d\n", len(successRepos))

	if len(failedRepos) > 0 {
		pterm.Warning.Printf("  Failed Repositories: %s\n", strings.Join(failedRepos, ", "))
	}

	pterm.Info.Printf("  Total Time: %v\n", elapsed)

	if len(failedRepos) > 0 {
		pterm.Warning.Printfln("⚠ Repository update completed with %d failure%s", len(failedRepos), pluralize(len(failedRepos), "", "s"))
		return fmt.Errorf("failed to update %d repositories", len(failedRepos))
	}

	pterm.Success.Printfln("✓ Repository update completed successfully")
	return nil
}

// Helper function to check if a repository name is in the list
func containsRepo(repos []string, name string) bool {
	for _, r := range repos {
		if r == name {
			return true
		}
	}
	return false
}

// Helper function to handle pluralization
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
