package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	statePushDir         string
	statePushForce       bool
	statePushBackup      bool
	statePushLock        bool
	statePushLockTimeout string
)

// statePushCmd represents the command to push local state to remote backend
var statePushCmd = &cobra.Command{
	Use:   "state-push",
	Short: "Push local state to remote backend",
	Long: `Push the local state file to the remote backend storage.
This command should be used with caution as it can overwrite remote state.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.StatePush(statePushDir, statePushForce, statePushBackup, statePushLock, statePushLockTimeout, useAI)

		if err != nil {
			terraform.ErrorHandler(err)
			return err
		}

		return nil
	},
	Example: `
    # Push local state to remote backend (will show diff first)
    smurf stf state-push

    # Force push without confirmation
    smurf stf state-push --force

    # Push without creating backup
    smurf stf state-push --no-backup

    # Push from specific directory
    smurf stf state-push --dir=path/to/terraform/code

    # Push with lock timeout
    smurf stf state-push --lock-timeout=60s

    # Push with AI assistance on errors
    smurf stf state-push --ai --force
    `,
}

func init() {
	statePushCmd.Flags().StringVar(&statePushDir, "dir", ".", "Specify the Terraform directory")
	statePushCmd.Flags().BoolVar(&statePushForce, "force", false, "Force push without confirmation")
	statePushCmd.Flags().BoolVar(&statePushBackup, "backup", true, "Create backup of remote state before pushing")
	statePushCmd.Flags().BoolVar(&statePushLock, "lock", true, "Lock the state file when pushing")
	statePushCmd.Flags().StringVar(&statePushLockTimeout, "lock-timeout", "0s", "Duration to retry acquiring a state lock")
	statePushCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	stfCmd.AddCommand(statePushCmd)
}
