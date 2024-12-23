package helm

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)


var statusCmd = &cobra.Command{
	Use:   "status [NAME]",
	Short: "Status of a Helm release.",
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

		err := helm.HelmStatus(releaseName, configs.Namespace)
		if err != nil {
			return fmt.Errorf(color.RedString("Helm status failed: %v", err))
		}
		return nil
	},
	Example: `
	smurf selm status my-release
	# In this example, it will fetch the status of 'my-release' in the 'default' namespace

	smurf selm status my-release -n my-namespace
	# In this example, it will fetch the status of 'my-release' in the 'my-namespace' namespace

	smurf selm status
	# In this example, it will read the release name from the config file and fetch its status
	`,
}

func init() {
	statusCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "Specify the namespace to get status of the Helm chart")
	selmCmd.AddCommand(statusCmd)
}
