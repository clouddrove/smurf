package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var graphDir string

// graphCmd defines a subcommand that generates a visual representation of the Terraform resources.
var graphCmd = &cobra.Command{
	Use:           "graph",
	Short:         "Generate a visual graph of Terraform resources",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.Graph(graphDir)
		if err != nil {
			os.Exit(1)
		}
		return nil
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
