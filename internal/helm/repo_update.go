package helm

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func Repo_Update(args []string) error {
	settings := helmCLI.New()

	// Create a more visually appealing spinner
	spinner, _ := pterm.DefaultSpinner.
		WithRemoveWhenDone(false).
		WithMessageStyle(pterm.NewStyle(pterm.FgLightCyan)).
		Start(pterm.LightCyan("Hang tight while we grab the latest from your chart repositories..."))

	defer func() {
		if r := recover(); r != nil {
			spinner.Fail("Unexpected error occurred: %v", r)
		}
	}()

	// Load repository file
	f, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		if os.IsNotExist(err) {
			spinner.Fail("No repositories found. You must add one first : ", err)
			return err
		}
		spinner.Fail("Failed to load repository config: ", err)
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
			spinner.Warning(fmt.Sprintf("Failed to create chart repository for %s: %v", cfg.Name, err))
			continue
		}
		repos = append(repos, r)
	}
	// Validate repositories
	if len(repos) == 0 {
		if updateAll {
			spinner.Fail("No repositories found. You must add one first.")
			return errors.New("no repositories found")
		}
		spinner.Fail("No repositories found matching the provided names")
		return errors.New("no repositories found matching the provided names")
	}

	// Prepare progress tracking
	totalRepos := len(repos)
	successRepos := make([]string, 0, totalRepos)
	failedRepos := make([]string, 0, totalRepos)

	spinner.UpdateText(pterm.LightCyan(fmt.Sprintf("Updating %d chart repositor%s...",
		totalRepos, pluralize(totalRepos, "y", "ies"))))

	start := time.Now()

	// Update repositories with enhanced logging
	for _, r := range repos {
		repoName := r.Config.Name
		spinner.UpdateText(pterm.Yellow(fmt.Sprintf("Fetching latest index for %s...", repoName)))

		if _, err := r.DownloadIndexFile(); err != nil {
			failedRepos = append(failedRepos, r.Config.Name)
			spinner.Warning(fmt.Sprintf("Failed to update repository %s: %v", r.Config.Name, err))
			continue
		}

		successRepos = append(successRepos, r.Config.Name)

		// Detailed success message for each repository
		pterm.Success.Println(fmt.Sprintf("Successfully got an update from the %s chart repository", r.Config.Name))
	}

	// Final summary
	elapsed := time.Since(start).Truncate(time.Millisecond)
	spinner.Success("Repository update completed successfully")

	// Print detailed summary
	pterm.Info.Println("\nUpdate Summary:")
	pterm.Info.Printf("• Total Repositories: %d\n", totalRepos)
	pterm.Info.Printf("• Successfully Updated: %v\n", len(successRepos))

	if len(failedRepos) > 0 {
		pterm.Warning.Printf("• Failed Repositories: %s\n", strings.Join(failedRepos, ", "))
	}

	pterm.Info.Printf("• Total Time: %v\n", elapsed.String())
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
