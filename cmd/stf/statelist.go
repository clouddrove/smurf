package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var listStateIds []string

// stateListCmd lists resources in the Terraform state
var stateListCmd = &cobra.Command{
	Use:   "list [address]",
	Short: "List resources in the Terraform state",
	Long: `List all resources or specific resources from the Terraform state.
If an address is provided, only resources matching that address will be listed.`,
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			listStateIds = append(listStateIds, args[0])
		}
		return terraform.StateList(stateDir, listStateIds, useAI)
	},
	Example: `
    # List all resources
    smurf stf state list
    
    # List specific resources
    smurf stf state list aws_instance.web
    
    # List resources from a specific module
    smurf stf state list module.vpc
    
    # List from custom directory
    smurf stf state list --dir=/path/to/terraform
    
    # Backup state before listing
    smurf stf state list --backup=backup.tfstate
    `,
}

func init() {
	stateCmd.AddCommand(stateListCmd)
}
