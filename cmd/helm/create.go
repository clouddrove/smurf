package helm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var createAuto bool

var createChartCmd = &cobra.Command{
	Use:   "create [NAME] [DIRECTORY]",
	Short: "Create a new Helm chart in the specified directory.",
	RunE: func(cmd *cobra.Command, args []string) error {

		if createAuto {

			data , err := configs.LoadConfig(configs.FileName)

			if err != nil {
				return err
			}

			if len(args) < 2 {
				args = append(args, data.Selm.ReleaseName, data.Selm.ChartName)
			}

			return helm.CreateChart(args[0], args[1])
		}

		return helm.CreateChart(args[0], args[1])
	},
	Example: `
	smurf selm create mychart ./mychart
	`,
}

func init() {
	createChartCmd.Flags().BoolVarP(&createAuto, "auto", "a", false, "Create Helm chart automatically")
	selmCmd.AddCommand(createChartCmd)
}
