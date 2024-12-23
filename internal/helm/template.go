package helm

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/strvals"
)

func HelmTemplate(releaseName, chartPath, namespace string, valuesFiles []string) error {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		color.Red("Failed to initialize action configuration: %v \n", err)
		return err
	}

	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Replace = true
	client.ClientOnly = true

	chart, err := loader.Load(chartPath)
	if err != nil {
		color.Red("Failed to load chart '%s': %v \n", chartPath, err)
		return err
	}

	vals := make(map[string]interface{})
	for _, f := range valuesFiles {
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			color.Red("Error reading values file '%s': %v \n", f, err)
			return err
		}
		for key, value := range additionalVals {
			vals[key] = value
		}
	}

	for _, set := range valuesFiles {
		if err := strvals.ParseInto(set, vals); err != nil {
			color.Red("Error parsing set values '%s': %v \n", set, err)
			return err
		}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Rendering Helm templates...\n")
	rel, err := client.Run(chart, vals)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to render templates: %v \n", err))
		return err
	}
	spinner.Success("Templates rendered successfully \n")

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Println(green(rel.Manifest))

	return nil
}
