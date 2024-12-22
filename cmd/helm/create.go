package helm

import (
	"errors"
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	directory string
	values    []string 
)

var createChartCmd = &cobra.Command{
	Use:   "create [NAME]",
	Short: "Create a new Helm chart in the specified directory.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string

		if len(args) >= 1 {
			name = args[0]
		}

		if name == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			name = data.Selm.ChartName
		}

		if name == "" {
			return errors.New(color.RedString("NAME must be provided either as an argument or in the config"))
		}

		if len(values) > 0 {
			fmt.Printf("Using values files: %v\n", values)
		}

		err := helm.CreateChart(name, directory)
		if err != nil {
			return fmt.Errorf(color.RedString("failed to create Helm chart: %v", err))
		}
		return nil
	},
	Example: `
smurf selm create mychart
# In this example, it will create 'mychart' in the current directory
smurf selm create
# In this example, it will create a chart with the name specified in the config in the current directory
`,
}

func init() {
	createChartCmd.Flags().StringArrayVarP(&values, "values", "f", []string{}, "Specify values in a YAML file")
	createChartCmd.Flags().StringVarP(&directory, "directory", "d", ".", "Specify the directory to create the Helm chart in")
	selmCmd.AddCommand(createChartCmd)
}
