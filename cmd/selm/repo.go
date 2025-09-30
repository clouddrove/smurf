package selm

import (
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo add|update [ARGS]",
	Short: "Add, update, or manage chart repositories",
	Long: `Add, update, or manage Helm chart repositories.
Common actions include adding a repository and updating repository indexes.`,
	SilenceUsage: true,
}

func init() {
	selmCmd.AddCommand(repoCmd)
}
