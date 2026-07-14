// selm/history.go
package selm

import (
	"fmt"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/spf13/cobra"
)

// historyCmd shows the revision history of a Helm release
var historyCmd = &cobra.Command{
	Use:          "history [RELEASE]",
	Short:        "Show revision history for a release",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !utils.ValidOutputFormat(outputFormat, "table", "json", "yaml") {
			return fmt.Errorf("invalid output format %q: must be one of table, json, yaml", outputFormat)
		}

		releaseName := args[0]
		namespace := configs.Namespace

		if namespace == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}
			namespace = data.Selm.Namespace
		}

		if namespace == "" {
			namespace = "default"
		}

		max, err := cmd.Flags().GetInt("max")
		if err != nil {
			return err
		}

		err = helm.HelmHistory(releaseName, namespace, max, outputFormat, useAI)
		if err != nil {
			return err
		}
		return nil
	},
	Example: `
# Show history for a release
smurf selm history my-release

# Show last 5 revisions
smurf selm history my-release --max 5

# Show history as a JSON document
smurf selm history my-release -o json
`,
}

func init() {
	historyCmd.Flags().Int("max", 256, "maximum number of revisions to show")
	historyCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "namespace of the release")
	historyCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")
	historyCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	historyCmd.ValidArgsFunction = completeReleaseNames
	_ = historyCmd.RegisterFlagCompletionFunc("namespace", completeNamespaces)
	_ = historyCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveDefault
	})

	selmCmd.AddCommand(historyCmd)
}
