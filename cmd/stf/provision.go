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
var provisionDir string // Define provisionDir variable

var provisionCmd = &cobra.Command{
	Use:          "provision",
	Short:        "Its the combination of init, plan, apply, output for Terraform",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := terraform.Init(provisionDir, upgrade); err != nil {
			return err
		}

		if err := terraform.Plan(varNameValue, varFile, provisionDir, planDestroy, planTarget, planRefresh, planState); err != nil {
			return err
		}

		if err := terraform.Apply(provisionApprove, varNameValue, varFile, lock, provisionDir, applyTarget, applyState); err != nil {
			return err
		}

		if err := terraform.Output(provisionDir); err != nil {
			return err
		}

		return nil
	},
	Example: `
	smurf stf provision
	smurf stf provision --dir=/path/to/terraform/files
	`,
}

func init() {
	provisionCmd.Flags().StringSliceVar(&varNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	provisionCmd.Flags().StringArrayVar(&varFile, "var-file", []string{}, "Specify a file containing variables")
	provisionCmd.Flags().BoolVar(&provisionApprove, "approve", true, "Skip interactive approval of plan before applying")
	provisionCmd.Flags().BoolVar(&lock, "lock", false, "Don't hold a state lock during the operation. This is dangerous if others might concurrently run commands against the same workspace.")
	provisionCmd.Flags().BoolVar(&upgrade, "upgrade", false, "Upgrade the Terraform modules and plugins to the latest versions")
	provisionCmd.Flags().StringVar(&provisionDir, "dir", "", "Specify the directory for Terraform operations") // Added flag
	stfCmd.AddCommand(provisionCmd)
}
