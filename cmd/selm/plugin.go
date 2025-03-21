package selm

import (
	"errors"
	"fmt"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var pluginName string

// pluginInstallCmd is a subcommand that installs a Helm plugin.
var pluginInstallCmd = &cobra.Command{
	Use:   "plugin install [PLUGIN]",
	Short: "Install a Helm plugin.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginName = args[0]
		if pluginName == "" {
			return errors.New(color.RedString("PLUGIN name must be provided"))
		}

		pterm.Info.Println(fmt.Sprintf("Installing Helm plugin: %s", pluginName))
		err := helm.HelmPluginInstall(pluginName)
		if err != nil {
			return errors.New(color.RedString("Helm plugin install failed: %v", err))
		}
		pterm.Success.Println("Helm plugin installed successfully.")
		return nil
	},
	Example: `
  smurf selm plugin https://github.com/helm/helm-secrets
  `,
}

func init() {
	selmCmd.AddCommand(pluginInstallCmd)
}
