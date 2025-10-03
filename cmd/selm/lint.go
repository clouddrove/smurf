package selm

import (
	"errors"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// lintCmd provides a subcommand to run a Helm lint check on a given chart.
// If no chart is specified on the command line, it attempts to read from the config file.
// Additionally, you can pass multiple values files to further customize the lint process.
var lintCmd = &cobra.Command{
	Use:          "lint [CHART]",
	Short:        "Lint a Helm chart.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var chartPath string

		if len(args) == 1 {
			chartPath = args[0]
		} else {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			chartPath = data.Selm.ChartName
			if chartPath == "" {
				pterm.Error.Println("CHART must be provided either as an argument or in the config")
				return errors.New("CHART must be provided either as an argument or in the config")
			}
		}

		err := helm.HelmLint(chartPath, configs.File)
		if err != nil {
			return err
		}
		return nil
	},
	Example: `
smurf selm lint ./mychart
smurf selm lint ./mychart -f ./my-chart/valus.yaml
smurf selm lint
# In the last example, it will read CHART from the config file
`,
}

func init() {
	lintCmd.Flags().StringArrayVarP(&configs.File, "values", "f", []string{}, "Specify values in a YAML file")
	selmCmd.AddCommand(lintCmd)
}
