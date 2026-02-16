package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var rmStateAddresses []string

// stateRmCmd removes items from the Terraform state
var stateRmCmd = &cobra.Command{
	Use:   "rm ADDRESS [ADDRESS...]",
	Short: "Remove items from the Terraform state",
	Long: `Remove one or more resources, modules, or instances from the Terraform state.
This does not destroy the real infrastructure, only removes them from the state file.`,
	SilenceUsage: true,
	Args:         cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rmStateAddresses = append(rmStateAddresses, args...)
		return terraform.StateRm(stateDir, rmStateAddresses, stateBackup, useAI)
	},
	Example: `
    # Remove a specific resource
    smurf stf state rm aws_instance.web
    
    # Remove multiple resources
    smurf stf state rm aws_instance.web aws_security_group.web
    
    # Remove all resources in a module
    smurf stf state rm module.vpc
    
    # With backup
    smurf stf state rm aws_instance.web --backup=backup.tfstate
    `,
}

func init() {
	stateCmd.AddCommand(stateRmCmd)
}
