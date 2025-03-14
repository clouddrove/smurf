package helm

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// HelmUpgrade performs a Helm upgrade operation for a specified release.
// It initializes the Helm action configuration, sets up the upgrade parameters,
// executes the upgrade, and then retrieves the status of the release post-upgrade.
// Detailed error logging is performed if any step fails.
func HelmUpgrade(releaseName, chartRef, namespace string, setValues []string, setLiteral []string, valuesFiles []string, createNamespace, atomic bool, timeout time.Duration, debug bool, repoURL string, version string) error {
	color.Green("Starting Helm Upgrade for release: %s \n", releaseName)
	
	// Handle namespace creation separately since Upgrade doesn't have CreateNamespace
	if createNamespace {
		if err := ensureNamespace(namespace, true); err != nil {
			logDetailedError("namespace creation", err, namespace, releaseName)
			return err
		}
	}

	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)
	
	logFn := func(format string, v ...interface{}) {
		if debug {
			fmt.Printf(format, v...)
			fmt.Println()
		}
	}
	
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logFn); err != nil {
		logDetailedError("helm action configuration", err, namespace, releaseName)
		return err
	}
	
	if actionConfig.KubeClient == nil {
		err := fmt.Errorf("KubeClient initialization failed")
		logDetailedError("kubeclient initialization", err, namespace, releaseName)
		return err
	}
	
	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Atomic = atomic
	client.Timeout = timeout
	client.Wait = true
	// Note: Upgrade action doesn't have CreateNamespace field, we handle it separately above
	
	// Use the LoadChart function to properly handle different chart sources
	chartObj, err := LoadChart(chartRef, repoURL, version, settings)
	if err != nil {
		logDetailedError("chart loading", err, namespace, releaseName)
		return err
	}
	
	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteral)
	if err != nil {
		logDetailedError("values loading", err, namespace, releaseName)
		return err
	}
	
	rel, err := client.Run(releaseName, chartObj, vals)
	if err != nil {
		logDetailedError("helm upgrade", err, namespace, releaseName)
		return err
	}
	
	if rel == nil {
		err := fmt.Errorf("no release object returned")
		logDetailedError("release object", err, namespace, releaseName)
		return err
	}
	
	color.Green("Upgrade Completed Successfully for release: %s \n", releaseName)
	printReleaseInfo(rel)
	printResourcesFromRelease(rel)
	
	err = monitorResources(rel, namespace, client.Timeout)
	if err != nil {
		logDetailedError("resource monitoring", err, namespace, releaseName)
		return err
	}
	
	color.Green("All resources for release '%s' after upgrade are ready and running.\n", releaseName)
	return nil
}