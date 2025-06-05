package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/chartutil"
)

// CreateChart generates a new Helm chart in the specified directory using Helm's chartutil package.
// It ensures that the target directory exists before creating the chart scaffolding.
// If successful, a success message is logged with the chart's location.
func CreateChart(chartName, saveDir string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Creating Helm chart '%s' in directory '%s'...", chartName, saveDir))
	defer spinner.Stop()

	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			pterm.Error.Printfln("Failed to create directory '%s': %v \n", saveDir, err)
			return err
		}
	}

	_, err := chartutil.Create(chartName, saveDir)
	pterm.Info.Printfln("Successfuly creates a new chart in a directory.")
	if err != nil {
		pterm.Error.Printfln("Failed to create chart '%s': %v \n", chartName, err)
		return err
	}
	homePathOfCreatedChart := filepath.Join(saveDir, chartName)
	spinner.Success(fmt.Sprintf("Chart '%s' created successfully at '%s'", chartName, homePathOfCreatedChart))
	return nil
}
