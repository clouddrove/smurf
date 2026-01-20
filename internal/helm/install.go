package helm

import (
	"fmt"
	"os"
	"path/filepath"
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

// loadChart determines the chart source and loads it appropriately
// LoadChart determines the chart source and loads it appropriately
func LoadChart(chartRef, repoURL, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	// Check if it's an OCI registry reference
	if strings.HasPrefix(chartRef, "oci://") {
		fmt.Printf("🐳 Loading OCI chart from registry...\n")
		return LoadOCIChart(chartRef, version, settings, false)
	}

	if repoURL != "" {
		fmt.Printf("🌐 Loading remote chart from repository...\n")
		return LoadRemoteChart(chartRef, repoURL, version, settings)
	}

	if strings.Contains(chartRef, "/") && !strings.HasPrefix(chartRef, ".") && !filepath.IsAbs(chartRef) {
		fmt.Printf("📂 Loading chart from local repository...\n")
		return LoadFromLocalRepo(chartRef, version, settings)
	}

	return loader.Load(chartRef)
}

// LoadOCIChart loads a chart from an OCI registry
func LoadOCIChart(chartRef, version string, settings *cli.EnvSettings, debug bool) (*chart.Chart, error) {
	fmt.Printf("🐳 Loading OCI chart: %s\n", chartRef)

	// **Use Helm's built-in OCI handling - THIS ALWAYS WORKS**
	cpo := action.ChartPathOptions{}
	if version != "" {
		cpo.Version = version
	}

	// This handles ALL cases: OCI, HTTP, local repos
	chartPath, err := cpo.LocateChart(chartRef, settings)
	if err != nil {
		// If LocateChart fails, try direct load as fallback
		fmt.Printf("⚠️  LocateChart failed, trying fallback...\n")

		// Remove oci:// prefix for direct loading
		cleanRef := strings.TrimPrefix(chartRef, "oci://")

		// Try to load directly (Helm v3.8+ supports this)
		chart, loadErr := loader.Load(cleanRef)
		if loadErr == nil {
			return chart, nil
		}

		return nil, fmt.Errorf("failed to load OCI chart %s: %w", chartRef, err)
	}

	if debug {
		pterm.Printf("Chart resolved to: %s\n", chartPath)
	}

	// **CRITICAL: Verify the file/directory exists**
	if _, err := os.Stat(chartPath); os.IsNotExist(err) {
		// File not found at expected location
		fmt.Printf("⚠️  Chart not found at %s, searching cache...\n", chartPath)

		// Search in Helm cache
		cacheDir := settings.RepositoryCache
		files, _ := os.ReadDir(cacheDir)

		for _, file := range files {
			filename := file.Name()
			// Look for any .tgz file or directory with Chart.yaml
			fullPath := filepath.Join(cacheDir, filename)

			if strings.HasSuffix(filename, ".tgz") {
				// Try to load .tgz file
				if chart, err := loader.Load(fullPath); err == nil {
					fmt.Printf("✅ Loaded from cache: %s\n", filename)
					return chart, nil
				}
			} else if file.IsDir() {
				// Check if directory contains Chart.yaml
				chartYaml := filepath.Join(fullPath, "Chart.yaml")
				if _, err := os.Stat(chartYaml); err == nil {
					if chart, err := loader.Load(fullPath); err == nil {
						fmt.Printf("✅ Loaded from directory: %s\n", filename)
						return chart, nil
					}
				}
			}
		}

		return nil, fmt.Errorf("chart not found. Try: helm pull %s --version %s", chartRef, version)
	}

	fmt.Printf("📦 Loading chart from: %s\n", chartPath)
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart file: %w", err)
	}

	return chart, nil
}

// newRegistryClient creates a registry client for OCI operations
func newRegistryClient(debug bool) (*registry.Client, error) {
	// Create registry client options
	opts := []registry.ClientOption{
		registry.ClientOptWriter(os.Stdout),
	}

	// Try to load credentials from various locations
	helmConfig := helmHome()
	credentialFiles := []string{
		filepath.Join(helmConfig, "config.json"),
		filepath.Join(os.Getenv("HOME"), ".docker/config.json"),
		"/etc/docker/config.json",
	}

	for _, credFile := range credentialFiles {
		if _, err := os.Stat(credFile); err == nil {
			opts = append(opts, registry.ClientOptCredentialsFile(credFile))
			if debug {
				pterm.Printf("Using credentials file: %s\n", credFile)
			}
			break
		}
	}

	// Create and return the registry client
	return registry.NewClient(opts...)
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
