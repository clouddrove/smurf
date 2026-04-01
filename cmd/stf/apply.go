package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var applyApprove bool
var applyVarNameValue []string
var applyVarFile []string
var applyLock bool
var applyDir string
var applyAutoApprove bool
var applyTarget []string
var applyState string
var applyPlanFile string
var useAI bool

// applyCmd defines a subcommand that applies the changes required to reach the desired state of Terraform Infrastructure.
var applyCmd = &cobra.Command{
	Use:          "apply",
	Short:        "Apply the changes required to reach the desired state of Terraform Infrastructure",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If a plan file is provided, we skip the approval prompt entirely
		// and auto-approve is ignored
		if applyPlanFile != "" {
			err := terraform.ApplyWithPlan(applyPlanFile, applyVarNameValue, applyVarFile, applyLock, applyDir, applyTarget, applyState, useAI)
			if err != nil {
				os.Exit(1)
			}
			return nil
		}
		// No plan file provided - use the regular apply flow with auto-approve option
		approve := applyAutoApprove
		err := terraform.Apply(approve, applyVarNameValue, applyVarFile, applyLock, applyDir, applyTarget, applyState, useAI)
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
	smurf stf apply

	# Specify variables
	smurf stf apply -var="region=us-west-2"

	# Skip approval prompt
	smurf stf apply --auto-approve

	# Apply using a plan file (automatically skips confirmation)
	smurf stf apply plan.out

	# Apply using a plan file with variables
	smurf stf apply plan.out -var="region=us-west-2"

	# Specify multiple variables
	smurf stf apply -var="region=us-west-2" -var="instance_type=t2.micro"

	# Specify a custom directory
	smurf stf apply --dir=/path/to/terraform/files

	# Target specific resources
	smurf stf apply --target=aws_instance.web
	smurf stf apply --target=module.vpc
	smurf stf apply --target=aws_instance.web --target=aws_security_group.web

	# Use custom state file
	smurf stf apply --state=/path/to/terraform.tfstate
	smurf stf apply -state=prod.tfstate
	`,
}

func init() {
	applyCmd.Flags().StringArrayVar(&applyVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	applyCmd.Flags().StringArrayVar(&applyVarFile, "var-file", []string{}, "Specify a file containing variables")
	applyCmd.Flags().BoolVar(&applyAutoApprove, "auto-approve", false, "Skip interactive approval of plan before applying")
	applyCmd.Flags().BoolVar(&applyLock, "lock", true, "Don't hold a state lock during the operation. This is dangerous if others might concurrently run commands against the same workspace.")
	applyCmd.Flags().StringVar(&applyDir, "dir", ".", "Specify the directory containing Terraform files")
	applyCmd.Flags().StringArrayVar(&applyTarget, "target", []string{}, "Target specific resources, modules, or resources in modules")
	applyCmd.Flags().StringVar(&applyState, "state", "", "Path to read and save the Terraform state")
	applyCmd.Flags().StringVar(&applyPlanFile, "plan", "", "Path to a plan file to apply (skips approval prompt)")
	applyCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	stfCmd.AddCommand(applyCmd)
}
