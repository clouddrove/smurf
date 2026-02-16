package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var pushForce bool

// statePushCmd pushes local state to remote backend
var statePushCmd = &cobra.Command{
	Use:   "push [state-file]",
	Short: "Push local state to remote backend",
	Long: `Push a local state file to the remote backend. This operation is DANGEROUS
as it can overwrite remote state. Use with extreme caution.`,
	SilenceUsage: true,
	Args:         cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		stateFile := ""
		if len(args) > 0 {
			stateFile = args[0]
		}
		return terraform.StatePush(stateDir, stateFile, pushForce, useAI)
	},
	Example: `
    # Push local state to remote
    smurf stf state push
    
    # Push specific state file
    smurf stf state push terraform.tfstate.backup
    
    # Force push (overwrite remote state)
    smurf stf state push --force
    
    # Force push specific file
    smurf stf state push terraform.tfstate.backup --force
    
    # Push from custom directory
    smurf stf state push --dir=/path/to/terraform
    `,
}

func init() {
	statePushCmd.Flags().BoolVar(&pushForce, "force", false, "Force overwriting remote state")
	stateCmd.AddCommand(statePushCmd)
}
