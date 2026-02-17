package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	statePullDir    string
	statePullBackup bool
)

// statePullCmd represents the command to pull and display remote Terraform state
var statePullCmd = &cobra.Command{
	Use:   "state-pull",
	Short: "Pull and display the current remote state",
	Long: `Fetch the current state from the remote backend and display it in JSON format.
This is useful for inspecting the state stored in remote backends like S3, GCS, or Terraform Cloud.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.StatePull(statePullDir, useAI)

		if err != nil {
			terraform.ErrorHandler(err)
			return err
		}

		return nil
	},
	Example: `
    # Pull and display remote state
    smurf stf state-pull

    # Pull state from specific directory
    smurf stf state-pull --dir=path/to/terraform/code

    # Pull state with AI assistance on errors
    smurf stf state-pull --ai --dir=prod/environment

    # Save pulled state to a file
    smurf stf state-pull > remote-state.json

    # Pretty print with jq (if installed)
    smurf stf state-pull | jq '.'
    `,
}

func init() {
	statePullCmd.Flags().StringVar(&statePullDir, "dir", ".", "Specify the Terraform directory")
	statePullCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	stfCmd.AddCommand(statePullCmd)
}
