package helm

import (
	"fmt"
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	provisionAuto      bool
	provisionNamespace string
)

var provisionCmd = &cobra.Command{
	Use:   "provision [RELEASE] [CHART]",
	Short: "Combination of install, upgrade, lint, and template for Helm",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if provisionAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			releaseName := data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			if len(args) < 2 {
				args = []string{releaseName, data.Selm.ChartName}
			}
			if provisionNamespace == "" {
				provisionNamespace = data.Selm.Namespace
			}

			return helm.HelmProvision(args[0], args[1], provisionNamespace)
		}

		if provisionNamespace == "" {
			provisionNamespace = "default"
		}
		return helm.HelmProvision(args[0], args[1], provisionNamespace)
	},
	Example: `
  smurf selm provision my-release ./mychart
  smurf selm provision --auto
  smurf selm provision --namespace custom-namespace my-release ./mychart
  `,
}

func init() {
	provisionCmd.Flags().BoolVarP(&provisionAuto, "auto", "a", false, "Provision Helm chart automatically")
	provisionCmd.Flags().StringVarP(&provisionNamespace, "namespace", "n", "", "Specify the namespace to provision the Helm chart")
	selmCmd.AddCommand(provisionCmd)
}
