package helm

import (
	"errors"
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var createAuto bool

var createChartCmd = &cobra.Command{
	Use:   "create [NAME] [DIRECTORY]",
	Short: "Create a new Helm chart in the specified directory.",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if createAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			releaseName := data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			chartName := data.Selm.ChartName
			if chartName == "" {
				return errors.New("chart name is not specified in the configuration")
			}

			if len(args) < 2 {
				args = []string{releaseName, chartName}
			} else {
				args[0] = releaseName
				args[1] = chartName
			}

			return helm.CreateChart(args[0], args[1])
		}

		if len(args) < 2 {
			return errors.New("requires exactly two arguments: [NAME] [DIRECTORY]")
		}

		return helm.CreateChart(args[0], args[1])
	},
	Example: `
smurf selm create mychart ./mychart
smurf selm create --auto
`,
}

func init() {
	createChartCmd.Flags().BoolVarP(&createAuto, "auto", "a", false, "Create Helm chart automatically")
	selmCmd.AddCommand(createChartCmd)
}
