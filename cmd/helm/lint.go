package helm

import (
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	autoLint bool
	lintFile []string
)

var lintCmd = &cobra.Command{
	Use:   "lint [CHART]",
	Short: "Lint a Helm chart.",
	RunE: func(cmd *cobra.Command, args []string) error {

		if autoLint {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			releaseName := data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			if len(args) < 1 {
				args = []string{releaseName}
			}

			return helm.HelmLint(args[0], lintFile)
		}

		chartPath := args[0]
		return helm.HelmLint(chartPath, lintFile)
	},
	Example: `
	smurf selm lint ./mychart
	`,
}

func init() {
	lintCmd.Flags().BoolVarP(&autoLint, "auto", "a", false, "Lint Helm chart automatically")
	lintCmd.Flags().StringArrayVarP(&lintFile, "values", "f", []string{}, "Specify values in a YAML file")
	selmCmd.AddCommand(lintCmd)
}
