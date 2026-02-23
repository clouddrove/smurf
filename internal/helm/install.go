package helm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

var oci string = "oci://"

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
	if strings.HasPrefix(chartRef, oci) {
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

	// Ensure cache directory exists
	if err := ensureHelmCacheDir(settings.RepositoryCache); err != nil {
		return nil, fmt.Errorf("failed to create helm cache directory: %w", err)
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
	pull.Untar = false // Keep as .tgz file
	pull.DestDir = settings.RepositoryCache

	// Run the pull command
	fmt.Printf("⬇️  Pulling OCI chart: %s...\n", chartRef)
	downloadedFile, err := pull.Run(chartRef)
	if err != nil {
		// The error might be about the file path, not the pull itself
		if debug {
			fmt.Printf("⚠️  Pull returned error but may have succeeded: %v\n", err)
			fmt.Printf("⚠️  Downloaded file path from pull.Run(): %s\n", downloadedFile)
		}

		// Continue to try loading the chart anyway
		return findAndLoadChartFromCache(chartRef, settings, debug)
	}

	if debug {
		fmt.Printf("✅ Pull reported success, downloaded to: %s\n", downloadedFile)
	}

	// Try to find and load the chart
	return findAndLoadChartFromCache(chartRef, settings, debug)
}

// Helper function to ensure helm cache directory exists
func ensureHelmCacheDir(cacheDir string) error {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		fmt.Printf("📁 Creating helm cache directory: %s\n", cacheDir)
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", cacheDir, err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check cache directory: %w", err)
	}
	return nil
}

// Helper function to find and load chart from cache
func findAndLoadChartFromCache(chartRef string, settings *cli.EnvSettings, debug bool) (*chart.Chart, error) {
	// Ensure directory exists (double-check)
	if err := ensureHelmCacheDir(settings.RepositoryCache); err != nil {
		return nil, err
	}

	// List all files in cache directory
	files, err := os.ReadDir(settings.RepositoryCache)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory %s: %w", settings.RepositoryCache, err)
	}

	if debug {
		fmt.Printf("📁 Searching for chart in cache directory: %s\n", settings.RepositoryCache)
		fmt.Printf("📁 Files found (%d):\n", len(files))
		for i, file := range files {
			info, _ := file.Info()
			fmt.Printf("  %d. %s (size: %d)\n", i+1, file.Name(), info.Size())
		}
	}

	// If no files found, try a different approach
	if len(files) == 0 {
		fmt.Println("⚠️  No files found in cache, attempting direct helm CLI pull...")
		return pullWithHelmCLI(chartRef, settings, debug)
	}

	// Extract chart name from OCI reference
	ref := strings.TrimPrefix(chartRef, oci)
	baseName := filepath.Base(ref)

	// Remove tag if present
	chartName := baseName
	if idx := strings.LastIndex(chartName, ":"); idx != -1 {
		chartName = chartName[:idx]
	}

	if debug {
		fmt.Printf("🔍 Looking for chart matching: %s\n", chartName)
	}

	// Look for .tgz files (most common)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".tgz") {
			fullPath := filepath.Join(settings.RepositoryCache, file.Name())

			if debug {
				fmt.Printf("   Trying .tgz file: %s\n", file.Name())
			}

			chartObj, err := loader.Load(fullPath)
			if err == nil {
				if debug {
					fmt.Printf("✅ Successfully loaded chart from: %s\n", fullPath)
				}
				return chartObj, nil
			}

			if debug {
				fmt.Printf("❌ Failed to load as chart: %v\n", err)
			}
		}
	}

	// Try any file (might not have .tgz extension)
	for _, file := range files {
		if !file.IsDir() {
			fullPath := filepath.Join(settings.RepositoryCache, file.Name())

			if debug {
				fmt.Printf("   Trying any file: %s\n", file.Name())
			}

			chartObj, err := loader.Load(fullPath)
			if err == nil {
				if debug {
					fmt.Printf("✅ Successfully loaded chart from: %s\n", fullPath)
				}
				return chartObj, nil
			}
		}
	}

	return nil, fmt.Errorf("no valid chart file found in cache directory: %s", settings.RepositoryCache)
}

// Fallback function using helm CLI directly
// Fallback function using helm CLI directly
func pullWithHelmCLI(chartRef string, settings *cli.EnvSettings, debug bool) (*chart.Chart, error) {
	fmt.Printf("🔄 Using helm CLI for OCI pull...\n")

	// Ensure cache directory exists
	if err := ensureHelmCacheDir(settings.RepositoryCache); err != nil {
		return nil, err
	}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "helm-oci-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Build helm command
	args := []string{"pull", chartRef, "--destination", tempDir}

	// Add version if specified
	if strings.Contains(chartRef, ":") {
		// Version might be in the chartRef itself
		fmt.Printf("📦 Chart reference includes version/tag\n")
	} else {
		// Parse version from chartRef or use default
		ref := strings.TrimPrefix(chartRef, oci)
		if idx := strings.LastIndex(ref, ":"); idx != -1 {
			version := ref[idx+1:]
			args = append(args, "--version", version)
		}
	}

	if debug {
		args = append(args, "--debug")
	}

	// Find helm binary using safe lookup
	helmPath, err := findHelmBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find helm binary: %w", err)
	}

	// Execute helm pull with absolute path
	cmd := exec.Command(helmPath, args...)

	// Use safe environment
	cmd.Env = getSafeEnvironment()

	// Enable OCI experimental feature
	cmd.Env = append(cmd.Env, "HELM_EXPERIMENTAL_OCI=1")

	// Handle GitHub Container Registry authentication
	if strings.Contains(chartRef, "ghcr.io") {
		fmt.Println("🔑 Detected GHCR registry")
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			fmt.Println("🔑 Using GITHUB_TOKEN for authentication")
			cmd.Env = append(cmd.Env, "GITHUB_TOKEN="+token)
		} else if token := os.Getenv("GH_TOKEN"); token != "" {
			fmt.Println("🔑 Using GH_TOKEN for authentication")
			cmd.Env = append(cmd.Env, "GH_TOKEN="+token)
		}
	}

	output, err := cmd.CombinedOutput()
	if debug {
		fmt.Printf("📋 Helm CLI output:\n%s\n", output)
	}

	if err != nil {
		return nil, fmt.Errorf("helm CLI pull failed: %w\nOutput: %s", err, output)
	}

	// Find and load the downloaded chart
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read temp directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			fullPath := filepath.Join(tempDir, file.Name())
			fmt.Printf("📦 Attempting to load: %s\n", file.Name())

			chartObj, err := loader.Load(fullPath)
			if err == nil {
				fmt.Printf("✅ Successfully loaded chart\n")

				// Copy to cache directory for future use
				cachePath := filepath.Join(settings.RepositoryCache, file.Name())
				if err := copyFile(fullPath, cachePath); err == nil && debug {
					fmt.Printf("📁 Copied to cache: %s\n", cachePath)
				}

				return chartObj, nil
			}

			if debug {
				fmt.Printf("❌ Failed to load: %v\n", err)
			}
		}
	}

	return nil, fmt.Errorf("no chart file found after helm pull")
}

// findHelmBinary safely locates the helm binary
func findHelmBinary() (string, error) {
	// Common helm installation paths across different platforms
	commonPaths := []string{
		// Linux (standard installations)
		"/usr/local/bin/helm",
		"/usr/bin/helm",
		"/bin/helm",

		// Linux (snap installations)
		"/snap/bin/helm",

		// macOS (Homebrew standard)
		"/usr/local/bin/helm",
		"/opt/homebrew/bin/helm", // Apple Silicon Homebrew
		"/usr/local/opt/helm/bin/helm",

		// macOS (MacPorts)
		"/opt/local/bin/helm",

		// Common user installations
		"/usr/local/helm/bin/helm",
		"/opt/helm/bin/helm",

		// GitHub Actions paths
		"/home/linuxbrew/.linuxbrew/bin/helm",     // Linuxbrew on GitHub Actions
		"/home/runner/.local/share/helm/bin/helm", // GitHub Actions runner
	}

	// First check common paths
	for _, path := range commonPaths {
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			// Check if it's executable
			if isExecutable(path) {
				return path, nil
			}
		}
	}

	// Fallback: Try PATH but only with safe directories
	safePathDirs := getSafePathDirectories()

	// Temporarily set PATH to safe directories for lookup
	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", strings.Join(safePathDirs, string(os.PathListSeparator)))
	defer os.Setenv("PATH", originalPath) // Restore original PATH

	// Use exec.LookPath with the safe PATH
	if path, err := exec.LookPath("helm"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("helm not found in common locations or safe PATH")
}

// getSafeEnvironment returns a sanitized environment
func getSafeEnvironment() []string {
	// Start with essential environment variables
	env := []string{}

	// Safe PATH for all platforms
	safePath := "PATH=" + strings.Join(getSafePathDirectories(), string(os.PathListSeparator))
	env = append(env, safePath)

	// Essential variables for Unix-like systems
	essentialVars := []string{
		"HOME",    // Needed for ~/.config, ~/.cache
		"USER",    // User information
		"LOGNAME", // Unix login name
		"SHELL",   // Shell path (sometimes used)
		"LANG",    // Language settings
		"LC_ALL",  // Locale settings
		"TMPDIR",  // Temporary directory
		"TMP",     // Temporary directory
		"TEMP",    // Temporary directory
		"TERM",    // Terminal type
	}

	// Copy essential variables if they exist
	for _, key := range essentialVars {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}

	// Copy variables that might be needed for authentication
	// (but filter out sensitive ones we handle explicitly)
	authRelated := []string{
		"DOCKER_CONFIG",   // Docker config location
		"KUBECONFIG",      // Kubernetes config
		"XDG_CONFIG_HOME", // XDG config directory
		"XDG_CACHE_HOME",  // XDG cache directory
		"XDG_DATA_HOME",   // XDG data directory
	}

	for _, key := range authRelated {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}

	// GitHub Actions specific variables
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		githubVars := []string{
			"GITHUB_WORKSPACE",
			"GITHUB_SHA",
			"GITHUB_REF",
			"RUNNER_TEMP",
			"RUNNER_WORKSPACE",
		}
		for _, key := range githubVars {
			if value := os.Getenv(key); value != "" {
				env = append(env, key+"="+value)
			}
		}
	}

	return env
}

// getSafePathDirectories returns safe, fixed directories for PATH
func getSafePathDirectories() []string {
	var userlocalbin string = "/usr/local/bin"
	// Common safe directories for all Unix-like systems
	commonDirs := []string{
		// Standard Linux/Unix directories
		"/usr/local/sbin",
		userlocalbin,
		"/usr/sbin",
		"/usr/bin",
		"/sbin",
		"/bin",

		// Homebrew (macOS and Linux)
		userlocalbin,
		"/opt/homebrew/bin",              // Apple Silicon Homebrew
		"/home/linuxbrew/.linuxbrew/bin", // Linuxbrew

		// Snap (Ubuntu/Linux)
		"/snap/bin",

		// MacPorts (macOS)
		"/opt/local/bin",
		"/opt/local/sbin",

		// System directories
		"/usr/local/games",
		"/usr/games",
	}

	// Remove duplicates and filter out non-existent directories
	seen := make(map[string]bool)
	var result []string

	for _, dir := range commonDirs {
		if !seen[dir] {
			seen[dir] = true
			// Check if directory exists and is not user-writable
			if isSafeDirectory(dir) {
				result = append(result, dir)
			}
		}
	}

	// Always include at least the bare minimum
	if len(result) == 0 {
		result = []string{userlocalbin, "/usr/bin", "/bin"}
	}

	return result
}

// isSafeDirectory checks if a directory exists and is not user-writable
func isSafeDirectory(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}

	if !info.IsDir() {
		return false
	}

	// Check if directory is writable by current user
	// In practice, system directories like /usr/bin are not user-writable
	// but we check anyway for extra safety
	if runtime.GOOS != "windows" {
		// On Unix-like systems, check if directory is world-writable
		mode := info.Mode()
		if mode&0002 != 0 { // World-writable bit is set
			return false
		}

		// Check if it's in user's home directory (potentially unsafe)
		if homeDir := os.Getenv("HOME"); homeDir != "" && strings.HasPrefix(dir, homeDir) {
			return false
		}

		// Check for other potentially unsafe locations
		unsafePrefixes := []string{"/tmp", "/var/tmp", "/dev/shm"}
		for _, prefix := range unsafePrefixes {
			if strings.HasPrefix(dir, prefix) {
				return false
			}
		}
	}

	return true
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a regular file and executable
	if runtime.GOOS != "windows" {
		// Unix-like systems
		return !info.IsDir() && info.Mode()&0111 != 0
	}

	// Windows: check file extension
	return !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".exe")
}

// Helper function to copy file
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
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
	helmPath := filepath.Join(userHome, ".helm")

	// Ensure directory exists
	if _, err := os.Stat(helmPath); os.IsNotExist(err) {
		os.MkdirAll(helmPath, 0755)
	}

	return helmPath
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
