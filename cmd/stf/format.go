package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var recursive bool

// formatCmd defines a subcommand that formats the Terraform Infrastructure.
var formatCmd = &cobra.Command{
	Use:   "format",
	Short: "Format the Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Format(recursive)
	},
	Example: `
	smurf stf format
	`,
}

func init() {
	formatCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Run the command recursively on all subdirectories .By default, only the given directory (or current directory) is processed.")
	stfCmd.AddCommand(formatCmd)
}
