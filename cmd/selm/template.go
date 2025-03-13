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

// Variables to store flag values
var repoURL string

var templateCmd = &cobra.Command{
	Use:   "template [RELEASE] [CHART]",
	Short: "Render chart templates",
	Args:  cobra.MaximumNArgs(2),
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
				return fmt.Errorf("failed to load config: %w", err)
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
				return errors.New(color.RedString("RELEASE and CHART must be provided either as arguments or in the config"))
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		err := helm.HelmTemplate(releaseName, chartPath, configs.Namespace, repoURL, configs.File)
		if err != nil {
			return fmt.Errorf(color.RedString("Helm template failed: %v", err))
		}
		return nil
	},
	Example: `
smurf selm template my-release ./mychart
# In this example, it will render templates for 'my-release' in './mychart' within the 'default' namespace

smurf selm template my-release mychart --repo https://charts.clouddrove.com/
# This will pull the chart from the specified Helm repository and render it.

smurf selm template my-release ./mychart -n my-namespace -f values.yaml
# In this example, it will render templates for 'my-release' in './mychart' within 'my-namespace' using specified values files
`,
}

func init() {
	templateCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "Specify the namespace to template the Helm chart")
	templateCmd.Flags().StringArrayVarP(&configs.File, "values", "f", []string{}, "Specify values in a YAML file")
	templateCmd.Flags().StringVarP(&repoURL, "repo", "r", "", "Specify Helm chart repository URL")
	selmCmd.AddCommand(templateCmd)
}