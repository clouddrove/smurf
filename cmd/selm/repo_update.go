package selm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var repoUpdateCmd = &cobra.Command{
	Use:          "update [REPO]...",
	Short:        "Update chart repositories",
	Long:         `Update all chart repositories or a specific one if provided.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var helmConfigDir string
		helmConfigDir = configs.HelmConfigDir
		return helm.Repo_Update(args, helmConfigDir)
	},
	Example: `   # Update all chart repositories
   smurf selm repo update
   
   # Update specific repositories
   smurf selm repo update prometheus stable`,
}

func init() {
	// Add helm-config flag for consistency
	repoUpdateCmd.Flags().StringVar(&configs.HelmConfigDir, "helm-config", "", "Helm configuration directory (default: $HELM_HOME or ~/.config/helm)")

	repoCmd.AddCommand(repoUpdateCmd)
}
