package helm

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// HelmInstall executes a full Helm install operation for the given release in the specified namespace.
// It creates the namespace (if required and permissioned), loads the chart from the provided path,
// merges the provided values (including --set overrides), and applies them in an atomic Helm install.
// If any step fails, it logs detailed information about the failure, including debug logs if enabled.
//
// Parameters:
//   - releaseName: The name for this particular Helm release.
//   - chartPath: Path to the Helm chart directory or packaged chart file.
//   - namespace: Kubernetes namespace in which to install the release.
//   - valuesFiles: A slice of file paths pointing to values.yaml files to merge.
//   - duration: Timeout duration for the install operation.
//   - Atomic: If true, the install will automatically roll back on failure.
//   - debug: Enables detailed debug output if true.
//   - setValues: Key-value pairs for setting or overriding chart values.
//
// On success, the function logs a success message and prints release resources.
// On failure, it logs contextual error details and returns the encountered error.
func HelmInstall(releaseName, chartPath, namespace string, valuesFiles []string, duration time.Duration, Atomic bool, debug bool, setValues []string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Starting Helm Install for release: %s", releaseName))
	defer spinner.Stop()

	if err := ensureNamespace(namespace, true); err != nil {
		logDetailedError("namespace creation", err, namespace, releaseName)
		return err
	}

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

	client := action.NewInstall(actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Atomic = Atomic
	client.Wait = true
	client.Timeout = duration
	client.CreateNamespace = true

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		color.Red("Chart Loading Failed: %s \n", chartPath)
		color.Red("Error: %v \n", err)
		color.Yellow("Try 'helm lint %s' to identify chart issues. \n", chartPath)
		return err
	}

	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues)
	if err != nil {
		logDetailedError("values loading", err, namespace, releaseName)
		return err
	}

	rel, err := client.Run(chartObj, vals)
	if err != nil {
		logDetailedError("helm install", err, namespace, releaseName)
		return err
	}

	if rel == nil {
		err := fmt.Errorf("no release object returned by Helm")
		logDetailedError("release object", err, namespace, releaseName)
		return err
	}

	spinner.Success(fmt.Sprintf("Installation Completed Successfully for release: %s \n", releaseName))
	printReleaseInfo(rel)

	printResourcesFromRelease(rel)

	err = monitorResources(rel, namespace, client.Timeout)
	if err != nil {
		logDetailedError("resource monitoring", err, namespace, releaseName)
		return err
	}

	color.Green("All resources for release '%s' are ready and running.\n", releaseName)
	return nil
}
