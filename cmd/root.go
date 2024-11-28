/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/spf13/cobra"
)

var originalHelpFunc func(*cobra.Command, []string)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "smurf",
	Short: "Smurf is a tool for automating common commands across Terraform, Docker, and more, streamlining DevOps workflows.",
	Long:  `Smurf is a command-line interface built with Cobra, designed to simplify and automate commands for essential tools like Terraform and Docker. It provides intuitive, unified commands to execute Terraform plans, Docker container management, and other DevOps tasks seamlessly from one interface. Whether you need to spin up environments, manage containers, or apply infrastructure as code, this CLI streamlines multi-tool operations, boosting productivity and reducing context-switching.
	If you are facing issues, unable to find a command, or need help, please create an issue at: https://github.com/clouddrove/smurf/issues
	`,
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
	originalHelpFunc = RootCmd.HelpFunc()

	RootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		displayBigText()
		originalHelpFunc(cmd, args)
	})

	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func displayBigText() {
	pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("S", pterm.FgCyan.ToStyle()),
		putils.LettersFromStringWithStyle("murf", pterm.FgLightMagenta.ToStyle())).
		Render()
}
