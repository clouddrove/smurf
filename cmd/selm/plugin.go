package selm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// pluginInstallCmd is a subcommand that installs multiple Helm plugins.
var pluginInstallCmd = &cobra.Command{
	Use:          "plugin install [PLUGINS]",
	Short:        "Install one or more Helm plugins (comma-separated).",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		pluginNames := strings.Join(args, ",")
		if pluginNames == "" {
			pterm.Error.Printfln("At least one PLUGIN name must be provided")
			return errors.New("at least one PLUGIN name must be provided")
		}

		pterm.Info.Println(fmt.Sprintf("Installing Helm plugins: %s", pluginNames))
		err := helm.HelmPluginInstall(pluginNames)
		if err != nil {
			pterm.Error.Printfln("Helm plugin install failed: %v", err)
			return fmt.Errorf("helm plugin install failed: %w", err)
		}
		pterm.Success.Println("Helm plugins installed successfully.")
		return nil
	},
	Example: `
  smurf selm plugin install https://github.com/helm/helm-secrets
  smurf selm plugin install https://github.com/helm/helm-secrets,https://github.com/helm/helm-diff
  `,
}

func init() {
	selmCmd.AddCommand(pluginInstallCmd)
}
