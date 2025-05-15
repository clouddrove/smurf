package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print detailed version information",
	Long:  `Print the version number of Smurf CLI along with build information`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func printVersion() {
	fmt.Printf("Smurf CLI version: %s\n", version)
	fmt.Printf("Git commit: %s\n", commit)
	fmt.Printf("Build date: %s\n", date)
}
