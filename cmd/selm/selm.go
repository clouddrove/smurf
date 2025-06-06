package selm

import (
	"github.com/clouddrove/smurf/cmd"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// selmCmd represents the 'selm' subcommand command
var selmCmd = &cobra.Command{
	Use:   "selm",
	Short: "Subcommand for Helm-related actions",
	Long:  `selm is a subcommand that groups various Helm-related actions under a single command.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.FgBlue.Printfln("Use 'smurf selm [command]' to run Helm-related actions")
	},
	Example: `smurf selm --help`,
}

func init() {
	cmd.RootCmd.AddCommand(selmCmd)
}
