package selm

import (
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var repoUpdateCmd = &cobra.Command{
	Use:   "update [REPO]...",
	Short: "Update chart repositories",
	Long:  `Update all chart repositories or a specific one if provided.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return helm.Repo_Update(args)
	},
	Example: `   # Update all chart repositories
   smurf selm repo update
   
   # Update specific repositories
   smurf selm repo update prometheus stable`,
}

func init() {
	repoCmd.AddCommand(repoUpdateCmd)
}
