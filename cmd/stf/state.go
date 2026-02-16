package stf

import (
	"github.com/spf13/cobra"
)

var (
	stateDir          string
	stateBackup       string
	stateIgnoreRemote bool
)

// stateCmd defines the parent command for Terraform state management
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Advanced state management for Terraform",
	Long: `Manage Terraform state with various subcommands including list, show, mv, rm, 
pull, push, and replace-provider. This command provides comprehensive state 
management capabilities similar to 'terraform state'.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no subcommand is provided, show help
		cmd.Help()
		return nil
	},
	Example: `
    # List all resources in state
    smurf stf state list
    
    # Show details of a specific resource
    smurf stf state show aws_instance.web
    
    # Move a resource in state
    smurf stf state mv aws_instance.old aws_instance.new
    
    # Remove a resource from state
    smurf stf state rm aws_instance.web
    
    # Pull remote state to local
    smurf stf state pull
    
    # Push local state to remote
    smurf stf state push
    
    # Replace provider in state
    smurf stf state replace-provider "registry.terraform.io/-/aws" "hashicorp/aws"
    `,
}

func init() {
	// Add common flags that apply to all state subcommands
	stateCmd.PersistentFlags().StringVar(&stateDir, "dir", ".", "Specify the directory containing Terraform files")
	stateCmd.PersistentFlags().StringVar(&stateBackup, "backup", "", "Path to backup the existing state file")
	stateCmd.PersistentFlags().BoolVar(&stateIgnoreRemote, "ignore-remote", false, "Ignore remote state configuration")

	stfCmd.AddCommand(stateCmd)
}
