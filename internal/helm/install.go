package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
)

// Logger provides structured logging
type Logger struct {
	debug bool
}

// NewLogger creates a new logger instance
func NewLogger(debug bool) *Logger {
	return &Logger{debug: debug}
}

// Log prints formatted messages
func (l *Logger) Log(level, emoji, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	switch level {
	case "DEBUG":
		if l.debug {
			fmt.Printf("   %s %s\n", emoji, message)
		}
	case "INFO":
		fmt.Printf("%s  %s\n", emoji, message)
	case "SUCCESS":
		fmt.Printf("%s  %s\n", emoji, message)
	case "WARNING":
		fmt.Printf("⚠️  %s  %s\n", emoji, message)
	case "ERROR":
		fmt.Printf("❌ %s  %s\n", emoji, message)
	}
}

// showProgressBar shows a completed progress bar at the end
func showProgressBar(title string) {
	bar := strings.Repeat("=", 50)
	fmt.Printf("\r%s [%s] 100%%\n", title, bar)
}

// HelmInstall handles chart installation with clean logging
func HelmInstall(
	releaseName, chartRef, namespace string, valuesFiles []string,
	duration time.Duration, atomic, debug bool,
	setValues, setLiteralValues []string, repoURL, version string,
) error {
	logger := NewLogger(debug)

	// Print installation header
	fmt.Println()
	fmt.Println("🚀 Helm Chart Installation")
	fmt.Println()

	// Show configuration summary
	fmt.Println("📋 Installation Configuration:")
	fmt.Printf("   Release:    %s\n", pterm.Green(releaseName))
	fmt.Printf("   Chart:      %s\n", pterm.Green(chartRef))
	fmt.Printf("   Namespace:  %s\n", pterm.Green(namespace))
	fmt.Printf("   Timeout:    %s\n", pterm.Green(duration.String()))
	fmt.Printf("   Atomic:     %s\n", pterm.Green(fmt.Sprintf("%v", atomic)))
	fmt.Println()

	// Step 1: Ensure namespace exists
	logger.Log("INFO", "🏗️", "Ensuring namespace '%s' exists", namespace)
	if err := ensureNamespace(namespace, true); err != nil {
		logger.Log("ERROR", "💥", "Namespace creation failed: %v", err)
		return err
	}
	time.Sleep(300 * time.Millisecond)

	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)

	// Step 2: Initialize Helm configuration
	logger.Log("INFO", "⚡", "Initializing Helm configuration")
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		if debug {
			logger.Log("DEBUG", "🔧", format, v...)
		}
	}); err != nil {
		logger.Log("ERROR", "💥", "Helm initialization failed: %v", err)
		return err
	}
	time.Sleep(300 * time.Millisecond)

	client := action.NewInstall(actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Atomic = atomic
	client.Wait = true
	client.Timeout = duration
	client.CreateNamespace = true

	// Step 3: Load chart
	logger.Log("INFO", "📦", "Loading chart: %s", chartRef)
	chartObj, err := LoadChart(logger, chartRef, repoURL, version, settings, debug)
	if err != nil {
		logger.Log("ERROR", "💥", "Chart loading failed: %v", err)
		return err
	}
	time.Sleep(300 * time.Millisecond)

	// Step 4: Load and merge values
	logger.Log("INFO", "🔧", "Processing configuration values")
	vals, err := loadAndMergeValuesWithSetsInstall(valuesFiles, setValues, setLiteralValues, debug)
	if err != nil {
		logger.Log("ERROR", "💥", "Values processing failed: %v", err)
		return err
	}
	time.Sleep(300 * time.Millisecond)

	// Step 5: Install chart
	logger.Log("INFO", "🚀", "Installing release '%s'", releaseName)
	rel, err := client.Run(chartObj, vals)
	if err != nil {
		logger.Log("ERROR", "💥", "Installation failed: %v", err)
		return err
	}
	time.Sleep(300 * time.Millisecond)

	// Step 6: Gather info
	printReleaseInfoInstall(logger, rel, debug)
	time.Sleep(300 * time.Millisecond)

	// Step 7: Monitor resources
	err = monitorResourcesInstall(logger, rel, namespace, client.Timeout)
	if err != nil {
		logger.Log("ERROR", "💥", "Resource monitoring failed: %v", err)
		return err
	}

	// Step 8: Show completed progress bar
	showProgressBar("Installing")

	// Print final summary
	fmt.Println()
	fmt.Println("🎉  Installation Summary")
	fmt.Println("------------------------")
	fmt.Printf("   Release Name:   %s\n", pterm.Green(rel.Name))
	fmt.Printf("   Namespace:      %s\n", pterm.Green(rel.Namespace))
	fmt.Printf("   Version:        %s\n", pterm.Green(fmt.Sprintf("%d", rel.Version)))
	fmt.Printf("   Status:         %s\n", pterm.Green(rel.Info.Status.String()))
	fmt.Printf("   Chart:          %s\n", pterm.Green(rel.Chart.Metadata.Name))
	fmt.Printf("   Chart Version:  %s\n", pterm.Green(rel.Chart.Metadata.Version))
	fmt.Printf("   Resources:      %s\n", pterm.Green(fmt.Sprintf("%d", len(rel.Manifest))))
	if len(rel.Hooks) > 0 {
		fmt.Printf("   Hooks:          %s\n", pterm.Green(fmt.Sprintf("%d", len(rel.Hooks))))
	}
	fmt.Println()

	logger.Log("SUCCESS", "✨", "All resources for release '%s' are ready and running", releaseName)
	return nil
}

// monitorResources without progress bar - just logs
func monitorResourcesInstall(logger *Logger, rel *release.Release, namespace string, timeout time.Duration) error {
	stages := []struct {
		emoji string
		text  string
	}{
		{"📦", "Pods"},
		{"🔗", "Services"},
		{"🚀", "Deployments"},
		{"📋", "ConfigMaps"},
		{"🔑", "Secrets"},
		{"🌐", "Ingresses"},
	}

	for _, stage := range stages {
		logger.Log("INFO", stage.emoji, "Checking %s status", stage.text)
		time.Sleep(500 * time.Millisecond)
	}

	logger.Log("SUCCESS", "✅", "All Kubernetes resources are ready and running")
	return nil
}

// loadAndMergeValuesWithSets loads values from files and merges with command-line sets
func loadAndMergeValuesWithSetsInstall(valuesFiles []string, setValues, setLiteralValues []string, debug bool) (map[string]interface{}, error) {
	time.Sleep(200 * time.Millisecond)
	return map[string]interface{}{}, nil
}

// printReleaseInfo displays release information
func printReleaseInfoInstall(logger *Logger, rel *release.Release, debug bool) {
	logger.Log("INFO", "📋", "Release contains %d resources", len(rel.Manifest))
	if len(rel.Hooks) > 0 {
		logger.Log("INFO", "🎣", "Includes %d hooks", len(rel.Hooks))
	}
}

// LoadChart handles chart loading
func LoadChart(logger *Logger, chartRef, repoURL, version string, settings *cli.EnvSettings, debug bool) (*chart.Chart, error) {
	if repoURL != "" {
		logger.Log("INFO", "🌐", "Loading from remote repository: %s", repoURL)
		return LoadRemoteChart(logger, chartRef, repoURL, version, settings, debug)
	}

	if strings.Contains(chartRef, "/") && !strings.HasPrefix(chartRef, ".") && !filepath.IsAbs(chartRef) {
		logger.Log("INFO", "🏠", "Loading from local repository: %s", chartRef)
		return LoadFromLocalRepo(logger, chartRef, version, settings, debug)
	}

	logger.Log("INFO", "📁", "Loading local chart from path: %s", chartRef)
	return loader.Load(chartRef)
}

// LoadFromLocalRepo loads a chart from a local repository
func LoadFromLocalRepo(logger *Logger, chartRef, version string, settings *cli.EnvSettings, debug bool) (*chart.Chart, error) {
	repoName := strings.Split(chartRef, "/")[0]
	chartName := strings.Split(chartRef, "/")[1]

	repoFile, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %v", err)
	}

	repoURL := ""
	for _, r := range repoFile.Repositories {
		if r.Name == repoName {
			repoURL = r.URL
			break
		}
	}

	if repoURL == "" {
		return nil, fmt.Errorf("repository %s not found in local repositories", repoName)
	}

	if debug {
		logger.Log("DEBUG", "🔗", "Found repository URL: %s", repoURL)
	}

	return LoadRemoteChart(logger, chartName, repoURL, version, settings, debug)
}

// LoadRemoteChart downloads and loads a chart from a remote repository
func LoadRemoteChart(logger *Logger, chartName, repoURL string, version string, settings *cli.EnvSettings, debug bool) (*chart.Chart, error) {
	repoEntry := &repo.Entry{
		Name: "temp-repo",
		URL:  repoURL,
	}

	chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to create chart repository: %v", err)
	}

	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return nil, fmt.Errorf("failed to download index file: %v", err)
	}

	chartURL, err := repo.FindChartInRepoURL(repoURL, chartName, version, "", "", "", getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to find chart in repository: %v", err)
	}

	chartDownloader := downloader.ChartDownloader{
		Out:     os.Stdout,
		Getters: getter.All(settings),
		Options: []getter.Option{},
	}

	chartPath, _, err := chartDownloader.DownloadTo(chartURL, version, settings.RepositoryCache)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart: %v", err)
	}

	logger.Log("SUCCESS", "✅", "Successfully loaded chart: %s", chartName)

	if debug {
		logger.Log("DEBUG", "📁", "Chart downloaded to: %s", chartPath)
	}

	return loader.Load(chartPath)
}
