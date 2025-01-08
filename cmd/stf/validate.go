package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// validateCmd defines a subcommand that validates the Terraform changes.
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate  Terraform changes",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Validate()
	},
	Example: `
	smurf stf validate
	`,
}

func init() {
	stfCmd.AddCommand(validateCmd)
}
