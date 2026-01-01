package helm

import (
	"fmt"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chartutil"
)

// HelmLint runs Helm's built-in linting on a specified chart directory or tarball,
// optionally merging values from given YAML files and parsing additional values,
// passed through the --set mechanism. Upon completion, it displays any detected
// linting messages, listing severity and location, or indicates if no issues were found
func HelmLint(chartPath string, fileValues []string, useAI bool) error {
	spinner, _ := pterm.DefaultSpinner.Start("Linting chart")
	defer spinner.Stop()

	client := action.NewLint()

	vals := make(map[string]interface{})
	for _, f := range fileValues {
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			pterm.Error.Printfln("Failed to read values file '%s': %v \n", f, err)
			ai.AIExplainError(useAI, err.Error())
			return err
		}
		for key, value := range additionalVals {
			vals[key] = value
		}
	}

	result := client.Run([]string{chartPath}, vals)
	if len(result.Messages) > 0 {
		for _, msg := range result.Messages {
			pterm.FgYellow.Printfln("Severity: %v", msg.Severity)
			pterm.FgYellow.Printfln("Path: %s", msg.Path)
			fmt.Println(msg)
			fmt.Println()
		}
		spinner.Info("Linting issues found \n")
	} else {
		pterm.FgGreen.Printfln("No linting issues found in the chart %s \n", chartPath)
		spinner.Success("Linting completed successfully \n")
	}

	pterm.Success.Printfln("Successfuly helm lint...")
	return nil
}
