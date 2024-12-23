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
