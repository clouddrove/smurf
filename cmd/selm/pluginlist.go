package selm

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// HelmPluginList retrieves and prints the list of installed Helm plugins.
func HelmPluginList() error {
	cmd := exec.Command("helm", "plugin", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		pterm.Error.Printfln("failed to list Helm plugins: %v", err)
		return fmt.Errorf("failed to list Helm plugins: %v", err)
	}

	pluginList := strings.TrimSpace(string(output))
	if pluginList == "" || pluginList == "NAME    VERSION DESCRIPTION" {
		pterm.Info.Printfln("No Helm plugins are currently installed.")
		return nil
	}

	pterm.Success.Println(pluginList)
	return nil
}

func runPluginList(cmd *cobra.Command, args []string) error {
	pterm.Info.Println("Listing Helm plugins...")
	if err := HelmPluginList(); err != nil {
		return err
	}

	pterm.Success.Printfln("Helm plugins listed successfully.")
	return nil
}

// pluginListCmd is a subcommand that lists installed Helm plugins.
var pluginListCmd = &cobra.Command{
	Use:          "list",
	Short:        "List all installed Helm plugins.",
	Example:      `smurf selm plugin list`,
	SilenceUsage: true,
	RunE:         runPluginList,
}

// pluginListDeprecatedCmd keeps the old top-level "plugin_list" command working
// as a hidden alias that forwards to "selm plugin list".
var pluginListDeprecatedCmd = &cobra.Command{
	Use:          "plugin_list",
	Short:        "List all installed Helm plugins.",
	Hidden:       true,
	Deprecated:   "use `smurf selm plugin list` instead",
	SilenceUsage: true,
	RunE:         runPluginList,
}

// Initialize subcommands
func init() {
	pluginCmd.AddCommand(pluginListCmd)
	selmCmd.AddCommand(pluginListDeprecatedCmd)
}
