package helm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/chartutil"
)

func CreateChart(chartName, saveDir string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Creating Helm chart '%s' in directory '%s'...", chartName, saveDir))
	defer spinner.Stop()

	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			color.Red("Failed to create directory '%s': %v \n", saveDir, err)
			return err
		}
	}

	_, err := chartutil.Create(chartName, saveDir)
	if err != nil {
		color.Red("Failed to create chart '%s': %v \n", chartName, err)
		return err
	}
	homePathOfCreatedChart := filepath.Join(saveDir, chartName)
	spinner.Success(fmt.Sprintf("Chart '%s' created successfully at '%s'", chartName, homePathOfCreatedChart))
	return nil
}
