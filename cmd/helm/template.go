package helm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	autoTemplate      bool
	templateNamespace string
    templateFiles     []string
)

var templateCmd = &cobra.Command{
	Use:   "template [RELEASE] [CHART]",
	Short: "Render chart templates ",
	RunE: func(cmd *cobra.Command, args []string) error {

		if autoTemplate {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if len(args) < 2 {
				args = append(args, data.Selm.ReleaseName, data.Selm.ChartName)
			}
			if templateNamespace == "" {
				templateNamespace = data.Selm.Namespace
			}

			return helm.HelmTemplate(args[0], args[1], templateNamespace, templateFiles)
		}

        if templateNamespace == "" {
            templateNamespace = "default"
        }

		return helm.HelmTemplate(args[0], args[1], templateNamespace, templateFiles)
	},
	Example: `
    smurf selm template my-release ./mychart
    `,
}

func init() {
	templateCmd.Flags().BoolVarP(&autoTemplate, "auto", "a", false, "Template Helm chart automatically")
	templateCmd.Flags().StringVarP(&templateNamespace, "namespace", "n", "", "Specify the namespace to template the Helm chart")
    templateCmd.Flags().StringArrayVarP(&templateFiles, "values", "f", []string{}, "Specify values in a YAML file")
	selmCmd.AddCommand(templateCmd)
}
