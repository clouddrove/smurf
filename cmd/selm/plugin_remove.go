package selm

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// HelmPluginUninstall uninstalls multiple Helm plugins.
func HelmPluginUninstall(plugins []string) error {
	for _, plugin := range plugins {
		plugin = strings.TrimSpace(plugin) // Trim spaces to avoid issues
		fmt.Printf("Uninstalling Helm plugin: %s...\n", plugin)

		// Execute helm plugin uninstall command
		cmd := exec.Command("helm", "plugin", "uninstall", plugin)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to uninstall Helm plugin %s: %v\n%s\n", plugin, err, string(output))
			continue
		}

		fmt.Printf("Helm plugin '%s' uninstalled successfully.\n", plugin)
	}

	return nil
}

// pluginUninstallCmd is a subcommand that uninstalls one or multiple Helm plugins.
var pluginUninstallCmd = &cobra.Command{
	Use:     "plugin_uninstall [plugin_name1,plugin_name2,...]",
	Short:   "Uninstall one or multiple Helm plugins.",
	Args:    cobra.ExactArgs(1),
	Example: `smurf selm plugin_uninstall my-plugin1,my-plugin2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginNames := strings.Split(args[0], ",")
		return HelmPluginUninstall(pluginNames)
	},
}

// Initialize the command
func init() {
	selmCmd.AddCommand(pluginUninstallCmd)
}
