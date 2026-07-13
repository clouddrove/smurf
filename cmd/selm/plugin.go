package selm

import (
	"github.com/spf13/cobra"
)

// pluginCmd is the parent command for managing Helm plugins.
// It groups install, list, and uninstall under a single "plugin" command,
// mirroring how "repo" groups add/update.
var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage Helm plugins",
	Long: `Manage Helm plugins.
Common actions include installing, listing, and uninstalling plugins.`,
	SilenceUsage: true,
}

func init() {
	selmCmd.AddCommand(pluginCmd)
}
