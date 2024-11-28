package docker

import (
	"fmt"

	"github.com/clouddrove/smurf/cmd"
	"github.com/spf13/cobra"
)

// sdkrCmd represents the 'sdkr' subcommand command
var sdkrCmd = &cobra.Command{
	Use:   "sdkr",
	Short: "Subcommand for Docker-related actions",
	Long:  `sdkr is a subcommand that groups various Docker-related actions under a single command.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use 'smurf sdkr [command]' to run Docker-related actions")
	},
	Example: `smurf sdkr --help`,
}

func init() {
	cmd.RootCmd.AddCommand(sdkrCmd)
}
