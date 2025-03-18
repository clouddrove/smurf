package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var applyApprove bool
var applyVarNameValue []string
var applyVarFile []string
var applyLock bool
var applyDir string
var applyAutoApprove bool

// applyCmd defines a subcommand that applies the changes required to reach the desired state of Terraform Infrastructure.
// applyCmd defines a subcommand that applies the changes required to reach the desired state of Terraform Infrastructure.
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the changes required to reach the desired state of Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {
		approve := applyApprove || applyAutoApprove
		return terraform.Apply(approve, applyVarNameValue, applyVarFile, applyLock, applyDir)
	},
	Example: `
	smurf stf apply

	# Specify variables
	smurf stf apply -var="region=us-west-2"

	# Skip approval prompt
	smurf stf apply --auto-approve

	# Specify multiple variables
	smurf stf apply -var="region=us-west-2" -var="instance_type=t2.micro"

	# Specify a custom directory
	smurf stf apply --dir=/path/to/terraform/files
	`,
}

func init() {
	applyCmd.Flags().StringArrayVar(&applyVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	applyCmd.Flags().StringArrayVar(&applyVarFile, "var-file", []string{}, "Specify a file containing variables")
	applyCmd.Flags().BoolVar(&applyApprove, "approve", false, "Skip interactive approval of plan before applying")
	applyCmd.Flags().BoolVar(&applyLock, "lock", true, "Don't hold a state lock during the operation. This is dangerous if others might concurrently run commands against the same workspace.")
	applyCmd.Flags().StringVar(&applyDir, "dir", ".", "Specify the directory containing Terraform files")
	applyCmd.Flags().BoolVar(&applyAutoApprove, "auto-approve", false, "Skip interactive approval of plan before applying")

	stfCmd.AddCommand(applyCmd)
}
