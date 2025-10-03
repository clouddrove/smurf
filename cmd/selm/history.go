// selm/history.go
package selm

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

// historyCmd shows the revision history of a Helm release
var historyCmd = &cobra.Command{
	Use:          "history [RELEASE]",
	Short:        "Show revision history for a release",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		err = helm.HelmHistory(releaseName, namespace, max)
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
`,
}

func init() {
	historyCmd.Flags().Int("max", 256, "maximum number of revisions to show")
	historyCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "namespace of the release")
	selmCmd.AddCommand(historyCmd)
}
