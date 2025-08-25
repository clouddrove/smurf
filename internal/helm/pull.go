package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func Pull(chartRef, version, destination string, untar bool, untarDir string,
	verify bool, keyring string, repoURL string, username string, password string,
	certFile string, keyFile string, caFile string, insecure bool, plainHttp bool,
	passCredentials bool, devel bool, prov bool, helmConfigDir string) error {

	pterm.Info.Printfln("Pulling chart: %s", chartRef)

	// Get Helm settings
	settings := getHelmSettingsPull(helmConfigDir)

	// Ensure destination directory exists
	if err := os.MkdirAll(destination, 0755); err != nil {
		pterm.Error.Printfln("✗ Failed to create destination directory: %v", err)
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Create action configuration
	actionConfig := new(action.Configuration)

	// Initialize with proper debug logging
	debugLog := func(format string, v ...interface{}) {
		pterm.Debug.Printfln(format, v...)
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		pterm.Error.Printfln("✗ Failed to initialize Helm action config: %v", err)
		return fmt.Errorf("failed to initialize Helm action config: %v", err)
	}

	// Create pull action
	pull := action.NewPullWithOpts(action.WithConfig(actionConfig))
	pull.Settings = settings
	pull.Version = version
	pull.DestDir = destination
	pull.Untar = untar
	pull.UntarDir = untarDir
	pull.Verify = verify
	pull.Keyring = keyring
	pull.RepoURL = repoURL
	pull.Username = username
	pull.Password = password
	pull.CertFile = certFile
	pull.KeyFile = keyFile
	pull.CaFile = caFile
	pull.InsecureSkipTLSverify = insecure
	pull.PlainHTTP = plainHttp
	pull.PassCredentialsAll = passCredentials
	pull.Devel = devel

	// Handle --prov flag (download provenance without verification)
	if prov && !verify {
		pull.Verify = false
		// We'll handle the provenance file check after download
	}

	// Execute the pull
	result, err := pull.Run(chartRef)
	if err != nil {
		pterm.Error.Printfln("✗ Failed to pull chart: %v", err)
		return fmt.Errorf("failed to pull chart: %v", err)
	}

	// Handle success message based on untar option
	if untar {
		pterm.Success.Printfln("✓ Successfully pulled and untarred chart: %s", filepath.Base(result))
	} else {
		pterm.Success.Printfln("✓ Successfully pulled chart: %s", filepath.Base(result))
	}

	pterm.Info.Printfln("  Location: %s", result)

	// Handle provenance file if --prov was specified without --verify
	if prov && !verify {
		provFile := result + ".prov"
		if _, err := os.Stat(provFile); err == nil {
			pterm.Success.Printfln("✓ Provenance file downloaded: %s", filepath.Base(provFile))
		} else {
			pterm.Info.Println("No provenance file found for this chart")
		}
	}

	// Show version if specified
	if version != "" {
		pterm.Info.Printfln("  Version: %s", version)
	}

	// Show verification status
	if verify {
		pterm.Success.Println("  Verification: ✓ Chart verified successfully")
	}

	return nil
}

// Helper function to get Helm settings
func getHelmSettingsPull(helmConfigDir string) *cli.EnvSettings {
	settings := cli.New()
	if helmConfigDir != "" {
		// Use custom helm config directory if provided
		settings.RepositoryConfig = filepath.Join(helmConfigDir, "repositories.yaml")
		settings.RepositoryCache = filepath.Join(helmConfigDir, "cache")
		settings.RegistryConfig = filepath.Join(helmConfigDir, "registry.json")
	}
	return settings
}
