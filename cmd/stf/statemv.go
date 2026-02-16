package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// stateMvCmd moves or renames items in the Terraform state
var stateMvCmd = &cobra.Command{
	Use:   "mv SOURCE DESTINATION",
	Short: "Move or rename an item in the Terraform state",
	Long: `Move or rename resources, modules, or instances in the Terraform state.
This is useful when you rename a resource or want to move it to a different module.`,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.StateMv(stateDir, args[0], args[1], stateBackup, useAI)
	},
	Example: `
    # Rename a resource
    smurf stf state mv aws_instance.web aws_instance.web_server
    
    # Move resource to a module
    smurf stf state mv aws_instance.web module.web.aws_instance.web
    
    # Move between modules
    smurf stf state mv module.vpc.aws_vpc.main module.network.aws_vpc.main
    
    # With backup
    smurf stf state mv aws_instance.web aws_instance.web_server --backup=backup.tfstate
    `,
}

func init() {
	stateCmd.AddCommand(stateMvCmd)
}
