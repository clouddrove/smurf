package helm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	statusNamespace string
	statusAuto      bool
)

var statusCmd = &cobra.Command{
	Use:   "status [NAME]",
	Short: "Status of a Helm release.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}
			args = append(args, data.ChartName)
			statusNamespace = data.Namespace
			return helm.HelmStatus(args[0], statusNamespace)
		}
		releaseName := args[0]
		if statusNamespace == "" {
			uninstallNamespace = "default"
		}
		return helm.HelmStatus(releaseName, statusNamespace)
	},
	Example: `
	smurf selm status my-release
	smurf selm status my-release -n my-namespace
	`,
}

func init() {
	statusCmd.Flags().StringVarP(&statusNamespace, "namespace", "n", "", "Specify the namespace to get status of the Helm chart ")
	statusCmd.Flags().BoolVarP(&statusAuto, "auto", "a", false, "Get status of Helm chart automatically")
	selmCmd.AddCommand(statusCmd)
}
