package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// formatCmd defines a subcommand that formats the Terraform Infrastructure.
var formatCmd = &cobra.Command{
	Use:   "format",
	Short: "Format the Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Format()
	},
	Example: `
	smurf stf format
	`,
}

func init() {
	stfCmd.AddCommand(formatCmd)
}
