package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	validateDir string // Variable for directory flag
)

// validateCmd defines a subcommand that validates the Terraform changes.
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate Terraform changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Validate(validateDir)
	},
	Example: `
 smurf stf validate
 smurf stf validate --dir=/path/to/terraform/files
`,
}

func init() {
	validateCmd.Flags().StringVar(&validateDir, "dir", "", "Directory containing Terraform files (default is current directory)")
	stfCmd.AddCommand(validateCmd)
}
