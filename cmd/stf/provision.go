package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var provisionApprove bool
var varNameValue []string
var varFile []string
var lock bool
var upgrade bool

// provisionCmd orchestrates multiple Terraform operations (init, plan, apply, output)
// in a sequential flow, grouping them into one streamlined command. After successful
// initialization, planning, and applying of changes, it retrieves the final outputs
// asynchronously and handles any errors accordingly.
var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Its the combination of init, plan, apply, output for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := terraform.Init(upgrade); err != nil {
			return err
		}

		if err := terraform.Plan(varNameValue, varFile); err != nil {
			return err
		}

		if err := terraform.Apply(provisionApprove, varNameValue, varFile, lock); err != nil {
			return err
		}

		if err := terraform.Output(); err != nil {
			return err
		}

		return nil
	},
	Example: `
	smurf stf provision
	`,
}

func init() {
	provisionCmd.Flags().StringSliceVar(&varNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	provisionCmd.Flags().StringArrayVar(&varFile, "var-file", []string{}, "Specify a file containing variables")
	provisionCmd.Flags().BoolVar(&provisionApprove, "approve", true, "Skip interactive approval of plan before applying")
	provisionCmd.Flags().BoolVar(&lock, "lock", false, "Don't hold a state lock during the operation. This is dangerous if others might concurrently run commands against the same workspace.")
	provisionCmd.Flags().BoolVar(&upgrade, "upgrade", false, "Upgrade the Terraform modules and plugins to the latest versions")
	stfCmd.AddCommand(provisionCmd)
}
