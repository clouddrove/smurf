package helm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	uninstallNamespace string
	uninstallAuto      bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [NAME]",
	Short: "Uninstall a Helm release.",
	RunE: func(cmd *cobra.Command, args []string) error {

		if uninstallAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}
			args = append(args, data.ChartName)
			uninstallNamespace = data.Namespace
			return helm.HelmUninstall(args[0], uninstallNamespace)
		}

		releaseName := args[0]
		if uninstallNamespace == "" {
			uninstallNamespace = "default"
		}
		return helm.HelmUninstall(releaseName, uninstallNamespace)
	},
	Example: `
	smurf selm uninstall my-release
	smurf selm uninstall my-release -n my-namespace
	`,
}

func init() {
	uninstallCmd.Flags().StringVarP(&uninstallNamespace, "namespace", "n", "", "Specify the namespace to uninstall the Helm chart ")
	uninstallCmd.Flags().BoolVarP(&uninstallAuto, "auto", "a", false, "Uninstall Helm chart automatically")
	selmCmd.AddCommand(uninstallCmd)
}
