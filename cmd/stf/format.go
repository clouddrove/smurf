package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var recursive bool

// formatCmd defines a subcommand that formats the Terraform Infrastructure.
var formatCmd = &cobra.Command{
	Use:          "format",
	Short:        "Format the Terraform Infrastructure",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.Format(recursive)
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
	smurf stf format
	`,
}

func init() {
	formatCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Run the command recursively on all subdirectories .By default, only the given directory (or current directory) is processed.")
	stfCmd.AddCommand(formatCmd)
}
