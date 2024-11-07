package terraform

import (
	"fmt"

	"github.com/clouddrove/smurf/cmd"
	"github.com/spf13/cobra"
)

// stfCmd represents the 'stf' command
var stfCmd = &cobra.Command{
	Use:   "stf",
	Short: "Subcommand for Terraform-related actions",
	Long:  `stf is a subcommand that groups various Terraform-related actions under a single command.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'smurf stf [command]' to run Terraform-related actions")
	},
}

func init() {
	// Add the 'stf' command as a subcommand of RootCmd
	cmd.RootCmd.AddCommand(stfCmd)
}
