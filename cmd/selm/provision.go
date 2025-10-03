package selm

import (
	"errors"
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// provisionCmd is a subcommand that orchestrates a more comprehensive Helm workflow
// by combining multiple steps like install, upgrade, lint, and template generation.
// It supports configurable arguments or fallback to values specified in the config file,
// as well as an optional custom namespace.
var provisionCmd = &cobra.Command{
	Use:          "provision [RELEASE] [CHART]",
	Short:        "Combination of install, upgrade, lint, and template for Helm",
	Args:         cobra.MaximumNArgs(2),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var releaseName, chartPath string

		if len(args) >= 1 {
			releaseName = args[0]
		}
		if len(args) >= 2 {
			chartPath = args[1]
		}

		if releaseName == "" || chartPath == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if releaseName == "" {
				releaseName = data.Selm.ReleaseName
				if releaseName == "" {
					releaseName = filepath.Base(data.Selm.ChartName)
				}
			}

			if chartPath == "" {
				chartPath = data.Selm.ChartName
			}

			if releaseName == "" || chartPath == "" {
				pterm.Error.Printfln("RELEASE and CHART must be provided either as arguments or in the config")
				return errors.New("RELEASE and CHART must be provided either as arguments or in the config")
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if releaseName == "" || chartPath == "" {
			pterm.Error.Printfln("RELEASE and CHART must be provided")
			return errors.New("RELEASE and CHART must be provided")
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		err := helm.HelmProvision(releaseName, chartPath, configs.Namespace)
		if err != nil {
			return err
		}
		return nil
	},
	Example: `
smurf selm provision my-release ./mychart
smurf selm provision
# In this example, it will read RELEASE and CHART from the config file
smurf selm provision my-release ./mychart -n custom-namespace
`,
}

func init() {
	provisionCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "Specify the namespace to provision the Helm chart")
	selmCmd.AddCommand(provisionCmd)
}
