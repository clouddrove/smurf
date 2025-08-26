package selm

import (
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var repoDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug Helm repository configuration",
	Long:  `Show debug information about Helm repository configuration paths and compatibility`,
	RunE: func(cmd *cobra.Command, args []string) error {
		helm.DebugHelmPaths()
		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoDebugCmd)
}
