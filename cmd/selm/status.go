package selm

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// statusCmd enables users to retrieve the status of a specified Helm release.
// It accepts an optional release name as a command argument or falls back to
// values in the config file. If neither is provided, it returns an error.
// Additionally, a custom namespace can be specified via a flag.
var statusCmd = &cobra.Command{
	Use:          "status [NAME]",
	Short:        "Status of a Helm release.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
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
				return errors.New(pterm.Error.Sprintfln("NAME must be provided either as an argument or in the config"))
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		err := helm.HelmStatus(releaseName, configs.Namespace, useAI)
		if err != nil {
			os.Exit(1)
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
	statusCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	selmCmd.AddCommand(statusCmd)
}
