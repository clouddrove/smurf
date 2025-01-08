package selm

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// uninstallCmd implements a Helm uninstall operation for a specified release.
// If no release name is provided on the command line, it attempts to read from
// the config file. Additionally, a custom namespace can be specified via flags.
// If none is provided, it defaults to "default".
var uninstallCmd = &cobra.Command{
	Use:   "uninstall [NAME]",
	Short: "Uninstall a Helm release.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var releaseName string

		if len(args) >= 1 {
			releaseName = args[0]
		}

		if releaseName == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			releaseName = data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			if releaseName == "" {
				return errors.New(color.RedString("NAME must be provided either as an argument or in the config"))
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		err := helm.HelmUninstall(releaseName, configs.Namespace)
		if err != nil {
			return fmt.Errorf(color.RedString("Helm uninstall failed: %v", err))
		}
		return nil
	},
	Example: `
smurf selm uninstall my-release
# In this example, it will uninstall 'my-release' from the 'default' namespace

smurf selm uninstall my-release -n my-namespace
# In this example, it will uninstall 'my-release' from the 'my-namespace' namespace

smurf selm uninstall
# In this example, it will read NAME from the config file and uninstall from the specified namespace or 'default' if not set
`,
}

func init() {
	uninstallCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "Specify the namespace to uninstall the Helm chart")
	selmCmd.AddCommand(uninstallCmd)
}
