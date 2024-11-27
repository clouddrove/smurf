package helm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	installNamespace string
	installAuto      bool
    installFiles     []string
)

var installCmd = &cobra.Command{
	Use:   "install [RELEASE] [CHART]",
	Short: "Install a Helm chart into a Kubernetes cluster.",
	RunE: func(cmd *cobra.Command, args []string) error {

		if installAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if installNamespace == "" {
				installNamespace = data.Selm.Namespace
			}

			if len(args) < 2 {
				args = append(args, data.Selm.ChartName, data.Selm.ReleaseName)
			}

            return helm.HelmInstall(args[0], args[1], installNamespace, installFiles)
		}

		releaseName := args[0]
		chartPath := args[1]
		if installNamespace == "" {
			installNamespace = "default"
		}
		return helm.HelmInstall(releaseName, chartPath, installNamespace, installFiles)
	},
	Example: `
    smurf selm install my-release ./mychart
    smurf selm install my-release ./mychart -n my-namespace
    smurf selm install my-release ./mychart -f values.yaml
    `,
}

func init() {
	installCmd.Flags().StringVarP(&installNamespace, "namespace", "n", "", "Specify the namespace to install the Helm chart")
	installCmd.Flags().BoolVarP(&installAuto, "auto", "a", false, "Install Helm chart automatically")
    installCmd.Flags().StringArrayVarP(&installFiles, "values", "f", []string{}, "Specify values in a YAML file")
	selmCmd.AddCommand(installCmd)
}
