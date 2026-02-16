package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// statePullCmd pulls remote state to local
var statePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull current state from remote backend",
	Long: `Retrieve the current state from the remote backend and output it to stdout.
This is useful for inspecting the state without modifying it.`,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.StatePull(stateDir, useAI)
	},
	Example: `
    # Pull remote state to stdout
    smurf stf state pull
    
    # Pull from custom directory
    smurf stf state pull --dir=/path/to/terraform
    
    # Save to file
    smurf stf state pull > terraform.tfstate.backup
    `,
}

func init() {
	stateCmd.AddCommand(statePullCmd)
}
