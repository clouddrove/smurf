package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	validateDir string // Variable for directory flag
)

// validateCmd defines a subcommand that validates the Terraform changes.
var validateCmd = &cobra.Command{
	Use:          "validate",
	Short:        "Validate Terraform changes",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.Validate(validateDir)
		if err != nil {
			os.Exit(1)
		}
		return nil
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
