package helm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	provisionAuto bool
	provisionNamespace string
)

var provisionCmd = &cobra.Command{
	Use:   "provision [RELEASE] [CHART]",
	Short: "Its the combination of install, upgrade, lint, template for Helm",
	RunE: func(cmd *cobra.Command, args []string) error {
		if provisionAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}
			args = append(args, data.ChartName, data.ChartDir)
			provisionNamespace = data.Namespace

			return helm.HelmProvision(args[0], args[1], provisionNamespace)
		}
		if provisionNamespace != "" {
			provisionNamespace = "default"
		}
		return helm.HelmProvision(args[0], args[1], provisionNamespace)
	},
	Example: `
	smurf selm provision my-release ./mychart
	`,
}

func init() {
	provisionCmd.Flags().BoolVarP(&provisionAuto, "auto", "a", false, "Provision Helm chart automatically")
	provisionCmd.Flags().StringVarP(&provisionNamespace, "namespace", "n", "", "Specify the namespace to provision the Helm chart")
	selmCmd.AddCommand(provisionCmd)
}
