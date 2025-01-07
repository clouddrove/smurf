package stf

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// provisionCmd orchestrates multiple Terraform operations (init, plan, apply, output)
// in a sequential flow, grouping them into one streamlined command. After successful
// initialization, planning, and applying of changes, it retrieves the final outputs 
// asynchronously and handles any errors accordingly.
var provisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Its the combination of init, plan, apply, output for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {
		approve := configs.CanApply
		if err := terraform.Init(); err != nil {
			return err
		}

		if err := terraform.Plan(varNameValue, varFile); err != nil {
			return err
		}

		if err := terraform.Apply(approve); err != nil {
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
	provisionCmd.Flags().StringVar(&varNameValue, "var", "", "Specify a variable in 'NAME=VALUE' format")
	provisionCmd.Flags().StringVar(&varFile, "var-file", "", "Specify a file containing variables")
	provisionCmd.Flags().BoolVar(&configs.CanApply, "approve", false, "Skip interactive approval of plan before applying")
	stfCmd.AddCommand(provisionCmd)
}
