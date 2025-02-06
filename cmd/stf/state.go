package stf

import (
    "github.com/clouddrove/smurf/internal/terraform"
    "github.com/spf13/cobra"
)

// stateListCmd represents the command to list resources in the Terraform state
var stateListCmd = &cobra.Command{
    Use:   "state-list",
    Short: "List resources in the Terraform state",
    RunE: func(cmd *cobra.Command, args []string) error {
        err := terraform.StateList()

        if err != nil {
            terraform.ErrorHandler(err)
            return err
        }

        return nil
    },
    Example: `
    # List all resources in state
    smurf stf state-list
    `,
}

func init() {
    stfCmd.AddCommand(stateListCmd)
}