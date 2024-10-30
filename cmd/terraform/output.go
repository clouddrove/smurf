package terraform


import (
	"github.com/clouddrove/smurf/cmd"
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var outputCmd = &cobra.Command{
	Use:   "output",
	Short: "Generate output for the current state of Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Output()
	},
}

func init() {
	cmd.RootCmd.AddCommand(outputCmd)
}
