package stf

import (
	"os"
	"time"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var recursive bool
var timeout time.Duration

// formatCmd defines a subcommand that formats the Terraform Infrastructure.
var formatCmd = &cobra.Command{
	Use:          "fmt",
	Short:        "Format the Terraform Infrastructure",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.Format(recursive, timeout)
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
	smurf stf fmt
	smurf stf fmt --timeout 30s
	smurf stf fmt --recursive --timeout 2m
	`,
}

func init() {
	formatCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Run the command recursively on all subdirectories. By default, only the given directory (or current directory) is processed.")
	formatCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0, "Timeout for the formatting process (e.g., 30s, 2m, 1h). Zero means no timeout.")
	stfCmd.AddCommand(formatCmd)
}
