package helm

import (
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint [CHART]",
	Short: "Lint a Helm chart.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartPath := args[0]
		return helm.HelmLint(chartPath)
	},
}

func init() {
	selmCmd.AddCommand(lintCmd)
}
