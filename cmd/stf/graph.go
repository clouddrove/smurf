package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// graphCmd represents the command to generate a visual representation of the Terraform resources
var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Generate a visual graph of Terraform resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Graph()
	},
	Example: `
    # Generate resource graph
    smurf stf graph
    `,
}

func init() {
	stfCmd.AddCommand(graphCmd)
}
