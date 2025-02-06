package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var applyApprove bool
var applyVarNameValue []string
var applyVarFile []string
var applyLock bool

// applyCmd defines a subcommand that applies the changes required to reach the desired state of Terraform Infrastructure.
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the changes required to reach the desired state of Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Apply(applyApprove, applyVarNameValue, applyVarFile, applyLock)
	},
	Example: `
	smurf stf apply

	# Specify variables
	smurf stf apply -var="region=us-west-2"

	# Specify multiple variables
	smurf stf apply -var="region=us-west-2" -var="instance_type=t2.micro"
	`,
}

func init() {
	applyCmd.Flags().StringArrayVar(&applyVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	applyCmd.Flags().StringArrayVar(&applyVarFile, "var-file", []string{}, "Specify a file containing variables")
	applyCmd.Flags().BoolVar(&applyApprove, "approve", false, "Skip interactive approval of plan before applying")
	applyCmd.Flags().BoolVar(&applyLock, "lock", true, "Don't hold a state lock during the operation. This is dangerous if others might concurrently run commands against the same workspace.")
	stfCmd.AddCommand(applyCmd)
}
