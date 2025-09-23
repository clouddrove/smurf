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
		pterm.Error.Printfln("Failed to initialize action configuration: %v", err)
		return err
	}

	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Replace = true
	client.ClientOnly = true
	client.ChartPathOptions.RepoURL = repoURL // Set repo URL if provided

	spinner, _ := pterm.DefaultSpinner.Start("Locating chart...")
	
	// ALWAYS use LocateChart to resolve the chart reference
	chartPathFinal, err := client.ChartPathOptions.LocateChart(chartPath, settings)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to locate chart '%s': %v", chartPath, err))
		return err
	}
	spinner.Success(fmt.Sprintf("Chart located: %s", chartPathFinal))

	spinner, _ = pterm.DefaultSpinner.Start("Loading chart...")
	chart, err := loader.Load(chartPathFinal)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to load chart: %v", err))
		return err
	}
	spinner.Success("Chart loaded successfully")

	// Process values files - CORRECTED VERSION
	vals := make(map[string]interface{})
	for _, f := range valuesFiles {
		spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Reading values file: %s", f))
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			spinner.Fail(fmt.Sprintf("Error reading values file '%s': %v", f, err))
			return err
		}
		
		// CORRECT: Merge the values maps properly
		// chartutil.CoalesceTables merges two maps[string]interface{}
		vals = chartutil.CoalesceTables(vals, additionalVals)
		
		spinner.Success(fmt.Sprintf("Values file processed: %s", f))
	}

	spinner, _ = pterm.DefaultSpinner.Start("Rendering Helm templates...")
	rel, err := client.Run(chart, vals)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to render templates: %v", err))
		return err
	}
	spinner.Success("Templates rendered successfully")

	pterm.FgGreen.Println(rel.Manifest)
	return nil
}