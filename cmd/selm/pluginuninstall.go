package selm

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// HelmPluginUninstall uninstalls multiple Helm plugins.
func HelmPluginUninstall(plugins []string) error {
	var failed []string

	for _, plugin := range plugins {
		plugin = strings.TrimSpace(plugin)
		if plugin == "" {
			continue
		}

		pterm.Info.Printfln("Uninstalling Helm plugin: %s...\n", plugin)

		cmd := exec.Command("helm", "plugin", "uninstall", plugin)
		output, err := cmd.CombinedOutput()
		if err != nil {
			pterm.Warning.Printfln("❌ Failed to uninstall Helm plugin '%s': %v\n%s\n", plugin, err, string(output))
			failed = append(failed, plugin)
			continue
		}

		pterm.Success.Printfln("✅ Helm plugin '%s' uninstalled successfully.\n", plugin)
	}

	if len(failed) > 0 {
		pterm.Error.Printfln("Some plugins failed to uninstall: %v", failed)
		return fmt.Errorf("some plugins failed to uninstall: %v", failed)
	}

	pterm.Success.Printfln("Helm plugin uninstall...")
	return nil
}

// pluginUninstallCmd is a subcommand that uninstalls one or multiple Helm plugins.
var pluginUninstallCmd = &cobra.Command{
	Use:          "plugin-uninstall [plugin1 plugin2 ...]",
	Short:        "Uninstall one or multiple Helm plugins.",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	Example:      `smurf selm plugin-uninstall my-plugin1 my-plugin2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return HelmPluginUninstall(args)
	},
}

// Initialize the command
func init() {
	selmCmd.AddCommand(pluginUninstallCmd)
}
