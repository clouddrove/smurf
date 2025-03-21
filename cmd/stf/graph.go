package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var graphDir string

// graphCmd defines a subcommand that generates a visual representation of the Terraform resources.
var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Generate a visual graph of Terraform resources",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Graph(graphDir)
	},
	Example: `
	smurf stf graph --dir <terraform-directory>
	smurf stf graph
	`,
}

func init() {
	graphCmd.Flags().StringVar(&graphDir, "dir", ".", "Specify the directory containing Terraform configurations")
	stfCmd.AddCommand(graphCmd)
}
