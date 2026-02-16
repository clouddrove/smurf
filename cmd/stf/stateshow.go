package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// stateShowCmd shows details of a specific resource in the Terraform state
var stateShowCmd = &cobra.Command{
	Use:   "show ADDRESS",
	Short: "Show details of a specific resource in the state",
	Long: `Display detailed attributes of a specific resource from the Terraform state.
The resource must be specified by its address in the state.`,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.StateShow(stateDir, args[0], useAI)
	},
	Example: `
    # Show details of a specific resource
    smurf stf state show aws_instance.web
    
    # Show module resource
    smurf stf state show module.vpc.aws_vpc.main
    
    # Show from custom directory
    smurf stf state show aws_instance.web --dir=/path/to/terraform
    
    # Backup state before showing
    smurf stf state show aws_instance.web --backup=backup.tfstate
    `,
}

func init() {
	stateCmd.AddCommand(stateShowCmd)
}
