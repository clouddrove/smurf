package stf

import (
	"fmt"

	"github.com/clouddrove/smurf/cmd"
	"github.com/spf13/cobra"
)

// stfCmd represents the 'stf' command
var stfCmd = &cobra.Command{
	Use:           "stf",
	Short:         "Subcommand for Terraform-related actions",
	Long:          `stf is a subcommand that groups various Terraform-related actions under a single command.`,
	SilenceErrors: true,
	// Without this, cobra treats an unknown subcommand as a positional
	// argument, runs the block below and exits 0. `smurf stf aply` would then
	// look like success to a pipeline that never ran anything.
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'smurf stf [command]' to run Terraform-related actions")
	},
	Example: `smurf stf --help`,
}

func init() {
	cmd.RootCmd.AddCommand(stfCmd)
}
