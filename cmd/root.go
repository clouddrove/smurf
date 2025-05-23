package cmd

import (
	"os"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/spf13/cobra"
)

var (
	version = "v1.0.0"  // will be overridden by ldflags during build
	commit  = "none"    // will be set by ldflags
	date    = "unknown" // will be set by ldflags
)

var originalHelpFunc func(*cobra.Command, []string)

// RootCmd represents the base command.
var RootCmd = &cobra.Command{
	Use:     "smurf",
	Version: version, // This enables --version flag automatically
	Short:   "Smurf is a tool for automating common commands across Terraform, Docker, and more",
	Long: `Smurf is a command-line interface built with Cobra, designed to simplify and automate commands for essential tools like Terraform and Docker. It provides intuitive, unified commands to execute Terraform plans, Docker container management, and other DevOps tasks seamlessly from one interface.
			If you are facing issues, unable to find a command, or need help, please create an issue at: https://github.com/clouddrove/smurf/issues`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
	Example: `smurf --help`,
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Set up custom help display
	originalHelpFunc = RootCmd.HelpFunc()
	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		displayBigText()
		originalHelpFunc(cmd, args)
	})

	// Add commands
	RootCmd.AddCommand(versionCmd)
}

// display smurf word
func displayBigText() {
	pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("S", pterm.FgCyan.ToStyle()),
		putils.LettersFromStringWithStyle("murf", pterm.FgLightMagenta.ToStyle()),
	).Render()
}
