package stf

import (
	"time"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var recursive bool
var timeout time.Duration
var formatDir string

// formatCmd defines a subcommand that formats the Terraform Infrastructure.
//
// The "format" alias exists for callers written against the wrong name. The stf
// group rejects an unknown subcommand, so `smurf stf format` used to exit 1
// having done nothing; before that guard it exited 0 having done nothing, which
// was worse. Keeping the alias means those callers format their files while
// they move to `fmt`.
var formatCmd = &cobra.Command{
	Use:          "fmt",
	Aliases:      []string{"format"},
	Short:        "Format the Terraform Infrastructure",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Format(formatDir, recursive, timeout)
	},
	Example: `
	smurf stf fmt
	smurf stf fmt --dir=infra/prod
	smurf stf fmt --timeout 30s
	smurf stf fmt --recursive --timeout 2m
	`,
}

func init() {
	formatCmd.Flags().StringVar(&formatDir, "dir", ".", "Specify the directory containing Terraform files")
	formatCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Run the command recursively on all subdirectories. By default, only the given directory (or current directory) is processed.")
	formatCmd.Flags().DurationVarP(&timeout, "timeout", "t", 0, "Timeout for the formatting process (e.g., 30s, 2m, 1h). Zero means no timeout.")
	stfCmd.AddCommand(formatCmd)
}
