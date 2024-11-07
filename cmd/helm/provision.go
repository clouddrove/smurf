package helm

import (
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var provisionCmd = &cobra.Command{
	Use:   "provision [RELEASE] [CHART]",
	Short: "Its the combination of install, upgrade, lint, template for Helm",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return helm.HelmProvision(args[0], args[1], "default")
	},
}

func init() {
	selmCmd.AddCommand(provisionCmd)
}
