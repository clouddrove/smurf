package helm

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
)

// HelmTemplate renders the Helm templates for a given chart, values files, and optionally a remote repo.
// HelmTemplate renders the Helm templates for a given chart, values files, and optionally a remote repo.
func HelmTemplate(releaseName, chartPath, namespace, repoURL string, valuesFiles []string) error {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		pterm.Error.Printfln("Failed to initialize action configuration: %v \n", err)
		return err
	}

	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Replace = true
	client.ClientOnly = true
	client.ChartPathOptions.RepoURL = repoURL // Set repo URL if provided

	var chartPathFinal string
	var err error

	if repoURL != "" {
		chartPathFinal, err = client.ChartPathOptions.LocateChart(chartPath, settings)
		if err != nil {
			pterm.Error.Printfln("Failed to locate chart in repository '%s': %v \n", repoURL, err)
			return err
		}
	} else {
		chartPathFinal = chartPath
	}

	chart, err := loader.Load(chartPathFinal)
	if err != nil {
		pterm.Error.Printfln("Failed to load chart '%s': %v \n", chartPathFinal, err)
		return err
	}

	vals := make(map[string]interface{})
	// Process values files
	for _, f := range valuesFiles {
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			pterm.Error.Printfln("Error reading values file '%s': %v \n", f, err)
			return err
		}
		for key, value := range additionalVals {
			vals[key] = value
		}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Rendering Helm templates...\n")
	rel, err := client.Run(chart, vals)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to render templates: %v \n", err))
		return err
	}
	spinner.Success("Templates rendered successfully \n")

	pterm.FgGreen.Println(rel.Manifest)
	return nil
}
