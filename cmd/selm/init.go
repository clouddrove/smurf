package selm

import (
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/spf13/cobra"
)

// defaultYamlContent defines the default structure for smurf.yaml (selm section)
var defaultYamlContent = `selm:
  releaseName: "Release Name"
  namespace: "Name Space"
  chartName: "Chart Name"
  revision: 0
`

// selmCreateCmd defines the "smurf selm init" command
var selmCreateCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default smurf.yaml file with selm configuration",
	Long: `This command generates a smurf.yaml file in the current working directory
with default selm configuration placeholders.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return utils.CreateYamlFile("smurf.yaml", defaultYamlContent)
	},
}

func init() {
	selmCmd.AddCommand(selmCreateCmd)
}
