package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version will be set during build via ldflags
var version = "v1.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	// Version flag  --version
	RootCmd.Flags().BoolP("version", "", false, "Show version information")

	// Preserve existing PersistentPreRun if it exists
	originalPersistentPreRun := RootCmd.PersistentPreRun

	RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Handle --version flag first
		if v, _ := cmd.Flags().GetBool("version"); v {
			fmt.Println(version)
			os.Exit(0)
		}

		// Chain to original PersistentPreRun if exists
		if originalPersistentPreRun != nil {
			originalPersistentPreRun(cmd, args)
		}
	}

	RootCmd.AddCommand(versionCmd)
}
