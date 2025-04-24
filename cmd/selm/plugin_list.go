package selm

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// HelmPluginList retrieves and prints the list of installed Helm plugins.
func HelmPluginList() error {
	cmd := exec.Command("helm", "plugin", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list Helm plugins: %v", err)
	}

	pluginList := strings.TrimSpace(string(output))

	if pluginList == "" || pluginList == "NAME    VERSION DESCRIPTION" {
		fmt.Println("No Helm plugins are currently installed.")
		return nil
	}

	fmt.Println(pluginList)
	return nil
}

// pluginListCmd is a subcommand that lists installed Helm plugins.
var pluginListCmd = &cobra.Command{
	Use:     "plugin_list",
	Short:   "List all installed Helm plugins.",
	Example: `smurf selm plugin_list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Listing Helm plugins...")
		err := HelmPluginList()
		if err != nil {
			return fmt.Errorf("Failed to list Helm plugins: %v", err)
		}
		fmt.Println("Helm plugins listed successfully.")
		return nil
	},
}

// Initialize subcommands
func init() {
	selmCmd.AddCommand(pluginListCmd)
}
