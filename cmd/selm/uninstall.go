package selm

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [NAME]",
	Short: "Uninstall a Helm release and all its resources",
	Long: `This command uninstalls a Helm release and ensures all associated Kubernetes resources
are properly deleted. It provides options for force deletion and timeout configuration.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var releaseName string

		if len(args) >= 1 {
			releaseName = args[0]
		}

		if releaseName == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			releaseName = data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			if releaseName == "" {
				pterm.Error.Printfln("NAME must be provided either as an argument or in the config")
				return errors.New("NAME must be provided either as an argument or in the config")
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		// Get flags
		forceDelete, _ := cmd.Flags().GetBool("force")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		disableHooks, _ := cmd.Flags().GetBool("no-hooks")
		cascade, _ := cmd.Flags().GetString("cascade")

		// Configure uninstall options
		opts := helm.UninstallOptions{
			ReleaseName:  releaseName,
			Namespace:    configs.Namespace,
			Force:        forceDelete,
			Timeout:      timeout,
			DisableHooks: disableHooks,
			Cascade:      cascade,
		}

		err := helm.HelmUninstall(opts)
		if err != nil {
			return err
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
	uninstallCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "Namespace of the release")
	uninstallCmd.Flags().Bool("force", false, "Force deletion of resources if Helm uninstall fails")
	uninstallCmd.Flags().Duration("timeout", 10*time.Minute, "Time to wait for deletion")
	uninstallCmd.Flags().Bool("no-hooks", false, "Prevent hooks from running during uninstall")
	uninstallCmd.Flags().String("cascade", "background", "Delete cascading policy (background, foreground, orphan)")
	selmCmd.AddCommand(uninstallCmd)
}
