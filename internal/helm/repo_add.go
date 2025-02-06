package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func Repo_Add(args []string, username, password, certFile, keyFile, caFile string) error {
	repoName := args[0]
	repoURL := args[1]

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Adding repo %s...", repoName))
	defer spinner.Stop()

	settings := helmCLI.New()

	// Create repository config directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(settings.RepositoryConfig), 0755); err != nil {
		return fmt.Errorf("failed to create repository config directory: %v", err)
	}

	// Initialize repository config
	var f *repo.File
	if _, err := os.Stat(settings.RepositoryConfig); err != nil {
		f = repo.NewFile()
	} else {
		f, err = repo.LoadFile(settings.RepositoryConfig)
		if err != nil {
			return fmt.Errorf("failed to load repository file: %v", err)
		}
	}

	// Check if repository already exists
	if f.Has(repoName) {
		spinner.Warning(fmt.Sprintf("Repository %s already exists", color.YellowString(repoName)))
		return fmt.Errorf("repository name (%s) already exists, please specify a different name", repoName)
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
		spinner.Fail(color.RedString("Failed to create chart repository"))
		return fmt.Errorf("failed to create chart repository: %v", err)
	}

	spinner.UpdateText("Downloading repository index...")

	// Download repository index
	start := time.Now()
	if _, err := r.DownloadIndexFile(); err != nil {
		spinner.Fail(color.RedString("Failed to get repository index"))
		return fmt.Errorf("looks like %q is not a valid chart repository or the URL cannot be reached: %v", repoURL, err)
	}

	elapsed := time.Since(start).Truncate(time.Millisecond)
	f.Add(entry)

	if err := f.WriteFile(settings.RepositoryConfig, 0644); err != nil {
		spinner.Fail(color.RedString("Failed to write repository config"))
		return fmt.Errorf("failed to write repository config: %v", err)
	}

	spinner.Success(fmt.Sprintf("Successfully added repo %s", color.GreenString(repoName)))
	pterm.Info.Printf("Repository has been added to your repositories file. (%v)\n", elapsed)
	return nil
}
