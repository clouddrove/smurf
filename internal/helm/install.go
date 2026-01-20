package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
)

// HelmInstall handles chart installation with three possible sources:
// 1. Remote repository URL (e.g., "https://prometheus-community.github.io/helm-charts")
// 2. Local repository reference (e.g., "prometheus-community/prometheus")
// 3. Local chart path (e.g., "./mychart")
func HelmInstall(
	releaseName, chartRef, namespace string, valuesFiles []string,
	duration time.Duration, atomic, debug bool,
	setValues, setLiteralValues []string, repoURL, version string,
	wait bool, useAI bool,
) error {
	fmt.Printf("📦 Ensuring namespace '%s' exists...\n", namespace)
	if err := ensureNamespace(namespace, true); err != nil {
		printErrorSummary("Namespace Preparation", releaseName, namespace, chartRef, err)
		return err
	}

	fmt.Printf("⚙️  Initializing Helm configuration...\n")
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)

	logFn := func(format string, v ...interface{}) {
		if debug {
			fmt.Printf("🔍 "+format+"\n", v...)
		}
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logFn); err != nil {
		printErrorSummary("Helm Configuration", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	fmt.Printf("🛠️  Setting up install action...\n")
	client := action.NewInstall(actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Atomic = atomic
	client.Wait = wait
	client.Timeout = duration
	client.CreateNamespace = true

	fmt.Printf("📊 Loading chart '%s'...\n", chartRef)
	var chartObj *chart.Chart
	var err error

	chartObj, err = LoadChart(chartRef, repoURL, version, settings)
	if err != nil {
		printErrorSummary("Chart Loading", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Load and merge values
	fmt.Printf("📝 Processing values and configurations...\n")
	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteralValues, debug)
	if err != nil {
		printErrorSummary("Values Processing", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	fmt.Printf("🚀 Installing release '%s'...\n", releaseName)

	// Run Helm install
	rel, err := client.Run(chartObj, vals)
	if err != nil {
		printReleaseResources(namespace, releaseName)
		printErrorSummary("Chart Installation", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// After Helm reports success, verify everything is actually healthy
	fmt.Printf("🔍 Verifying installation health...\n")
	if err := verifyInstallationHealth(namespace, releaseName, duration, debug); err != nil {
		printReleaseResources(namespace, releaseName)
		printErrorSummary("Chart Installation", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	// Only if everything is healthy, print success
	return handleInstallationSuccess(rel, namespace)
}

// LoadChart determines the chart source and loads it appropriately
func LoadChart(chartRef, repoURL, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	// Check if it's an OCI registry reference
	if strings.HasPrefix(chartRef, "oci://") {
		fmt.Printf("🐳 Loading OCI chart from registry...\n")
		return LoadOCIChart(chartRef, version, settings, false) // You might want to make debug configurable
	}

	if repoURL != "" {
		fmt.Printf("🌐 Loading remote chart from repository...\n")
		return LoadRemoteChart(chartRef, repoURL, version, settings)
	}

	if strings.Contains(chartRef, "/") && !strings.HasPrefix(chartRef, ".") && !filepath.IsAbs(chartRef) {
		fmt.Printf("📂 Loading chart from local repository...\n")
		return LoadFromLocalRepo(chartRef, version, settings)
	}

	// Handle local chart file or directory
	return loader.Load(chartRef)
}

// LoadOCIChart loads a chart from an OCI registry
func LoadOCIChart(chartRef, version string, settings *cli.EnvSettings, debug bool) (*chart.Chart, error) {
	if debug {
		pterm.Printf("Loading OCI chart: %s (version: %s)\n", chartRef, version)
	}

	// Create registry client
	registryClient, err := newRegistryClient(debug)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}

	// Create action configuration with registry client
	actionConfig := &action.Configuration{
		RegistryClient: registryClient,
	}

	// Create pull action
	pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
	pull.Settings = settings
	pull.Version = version
	pull.Untar = false // DO NOT untar - we want the .tgz file
	pull.DestDir = settings.RepositoryCache

	// Run the pull command
	fmt.Printf("⬇️  Pulling OCI chart: %s...\n", chartRef)
	downloadedFile, err := pull.Run(chartRef)
	if err != nil {
		return nil, fmt.Errorf("failed to pull OCI chart: %w", err)
	}

	// The downloadedFile path is returned by pull.Run()
	// In newer Helm versions, it returns the actual file path
	chartPath := downloadedFile

	// If the path is not absolute, make it relative to cache dir
	if !filepath.IsAbs(chartPath) {
		chartPath = filepath.Join(settings.RepositoryCache, chartPath)
	}

	// Verify the file exists
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		// Try to find the chart file in the cache directory
		foundPath := findChartFileInCache(settings.RepositoryCache, chartRef, debug)
		if foundPath != "" {
			chartPath = foundPath
		} else {
			// Debug: list all files in cache
			if debug {
				listCacheFiles(settings.RepositoryCache)
			}
			return nil, fmt.Errorf("chart file not found at: %s", chartPath)
		}
	}

	if debug {
		pterm.Printf("OCI chart located at: %s\n", chartPath)
	}

	fmt.Printf("📦 Loading OCI chart into memory...\n")
	return loader.Load(chartPath)
}

// Helper function to find chart file in cache
func findChartFileInCache(cacheDir, chartRef string, debug bool) string {
	// Extract chart name from OCI reference
	ref := strings.TrimPrefix(chartRef, "oci://")
	chartName := filepath.Base(ref)

	// Remove tag if present
	if idx := strings.LastIndex(chartName, ":"); idx != -1 {
		chartName = chartName[:idx]
	}

	// List files in cache directory
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		return ""
	}

	// Look for files matching our chart
	var possibleMatches []string
	for _, file := range files {
		filename := file.Name()

		// Check if file contains chart name and ends with .tgz
		if strings.Contains(filename, chartName) && strings.HasSuffix(filename, ".tgz") {
			fullPath := filepath.Join(cacheDir, filename)
			possibleMatches = append(possibleMatches, fullPath)

			if debug {
				pterm.Printf("Found possible chart file: %s\n", fullPath)
			}
		}
	}

	// Return the most recent file if multiple found
	if len(possibleMatches) > 0 {
		// Sort by modification time (newest first)
		sort.Slice(possibleMatches, func(i, j int) bool {
			infoI, _ := os.Stat(possibleMatches[i])
			infoJ, _ := os.Stat(possibleMatches[j])
			return infoI.ModTime().After(infoJ.ModTime())
		})
		return possibleMatches[0]
	}

	return ""
}

// Helper to list cache files for debugging
func listCacheFiles(cacheDir string) {
	fmt.Println("📁 Contents of cache directory:")
	files, err := os.ReadDir(cacheDir)
	if err != nil {
		fmt.Printf("Error reading cache directory: %v\n", err)
		return
	}

	for _, file := range files {
		info, _ := file.Info()
		fmt.Printf("  - %s (size: %d, mod: %s)\n",
			file.Name(),
			info.Size(),
			info.ModTime().Format("15:04:05"))
	}
}

// newRegistryClient creates a registry client for OCI operations
func newRegistryClient(debug bool) (*registry.Client, error) {
	// Create registry client options
	opts := []registry.ClientOption{
		registry.ClientOptWriter(os.Stderr), // Use stderr for debug output
		registry.ClientOptDebug(debug),
	}

	// Try multiple credential sources
	helmConfig := helmHome()

	// Check for Docker config in multiple locations
	possibleCredFiles := []string{
		filepath.Join(helmConfig, "config.json"),
		filepath.Join(os.Getenv("HOME"), ".docker/config.json"),
		"/etc/docker/config.json",
		filepath.Join(os.Getenv("HOME"), ".helm/registry/config.json"),
	}

	for _, credFile := range possibleCredFiles {
		if _, err := os.Stat(credFile); err == nil {
			opts = append(opts, registry.ClientOptCredentialsFile(credFile))
			if debug {
				pterm.Printf("Using credentials file: %s\n", credFile)
			}
			break
		}
	}

	// Also check for environment variables
	if auth := os.Getenv("HELM_REGISTRY_CONFIG"); auth != "" {
		opts = append(opts, registry.ClientOptCredentialsFile(auth))
	}

	// Create and return the registry client
	client, err := registry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}

	return client, nil
}

// helmHome gets the Helm home directory
func helmHome() string {
	if home := os.Getenv("HELM_HOME"); home != "" {
		return home
	}
	if home := os.Getenv("HELM_CONFIG_HOME"); home != "" {
		return home
	}
	userHome, _ := os.UserHomeDir()
	return filepath.Join(userHome, ".helm")
}

// loadFromLocalRepo loads a chart from a local repository
func LoadFromLocalRepo(chartRef, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	repoName := strings.Split(chartRef, "/")[0]
	chartName := strings.Split(chartRef, "/")[1]

	repoFile, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		pterm.Error.Printfln("failed to load repository file: %v", err)
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
		pterm.Error.Printfln("repository %s not found in local repositories", repoName)
		return nil, fmt.Errorf("repository %s not found in local repositories", repoName)
	}

	return LoadRemoteChart(chartName, repoURL, version, settings)
}

// loadRemoteChart downloads and loads a chart from a remote repository
func LoadRemoteChart(chartName, repoURL string, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	fmt.Printf("🔗 Connecting to repository %s...\n", repoURL)
	repoEntry := &repo.Entry{
		Name: "temp-repo",
		URL:  repoURL,
	}

	chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to create chart repository: %v", err)
	}

	fmt.Printf("📥 Downloading repository index...\n")
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return nil, fmt.Errorf("failed to download index file: %v", err)
	}

	fmt.Printf("🔍 Finding chart %s in repository...\n", chartName)
	chartURL, err := repo.FindChartInRepoURL(repoURL, chartName, version, "", "", "", getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to find chart in repository: %v", err)
	}

	fmt.Printf("⬇️  Downloading chart...\n")
	chartDownloader := downloader.ChartDownloader{
		Out:     os.Stdout,
		Getters: getter.All(settings),
		Options: []getter.Option{},
	}

	chartPath, _, err := chartDownloader.DownloadTo(chartURL, version, settings.RepositoryCache)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart: %v", err)
	}

	fmt.Printf("📦 Loading chart into memory...\n")
	return loader.Load(chartPath)
}
