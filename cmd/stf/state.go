package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var stateListDir string

// stateListCmd represents the command to list resources in the Terraform state
var stateListCmd = &cobra.Command{
	Use:          "state-list",
	Short:        "List resources in the Terraform state",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.StateList(stateListDir, useAI)

		if err != nil {
			terraform.ErrorHandler(err)
			return err
		}

		return nil
	},
	Example: `
    # List all resources in state
    smurf stf state-list

    # List resources in a specific directory
    smurf stf state-list --dir=path/to/terraform/code
    `,
}

func init() {
	stateListCmd.Flags().StringVar(&stateListDir, "dir", ".", "Specify the Terraform directory")
	stateListCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	stfCmd.AddCommand(stateListCmd)
}
