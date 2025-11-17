package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var planVarNameValue []string
var planVarFile []string
var planDir string
var planDestroy bool
var planTarget []string
var planRefresh bool
var planState string // Added state flag

// planCmd defines a subcommand that generates and shows an execution plan for Terraform
var planCmd = &cobra.Command{
	Use:          "plan",
	Short:        "Generate and show an execution plan for Terraform",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		terraform.Plan(planVarNameValue, planVarFile, planDir, planDestroy, planTarget, planRefresh, planState)
		return nil
	},
	Example: `
    smurf stf plan

    # Specify variables
    smurf stf plan -var="region=us-west-2"

    # Specify multiple variables
    smurf stf plan -var="region=us-west-2" -var="instance_type=t2.micro"

    # Specify a custom directory
    smurf stf plan --dir=/path/to/terraform/files

    # Plan for destroy
    smurf stf plan --destroy

    # Target specific resources
    smurf stf plan --target=aws_instance.web
    smurf stf plan --target=module.vpc
    smurf stf plan --target=aws_instance.web --target=aws_security_group.web

    # Skip refresh
    smurf stf plan --refresh=false

    # Use custom state file
    smurf stf plan --state=/path/to/terraform.tfstate
    smurf stf plan -state=prod.tfstate

    # Combine with other flags
    smurf stf plan --target=aws_instance.web --destroy --var="instance_type=t2.micro" --refresh=false --state=prod.tfstate
    `,
}

func init() {
	planCmd.Flags().StringArrayVar(&planVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	planCmd.Flags().StringArrayVar(&planVarFile, "var-file", []string{}, "Specify a file containing variables")
	planCmd.Flags().StringVar(&planDir, "dir", ".", "Specify the directory containing Terraform files")
	planCmd.Flags().BoolVar(&planDestroy, "destroy", false, "Generate a destroy plan")
	planCmd.Flags().StringArrayVar(&planTarget, "target", []string{}, "Target specific resources, modules, or resources in modules")
	planCmd.Flags().BoolVar(&planRefresh, "refresh", true, "Update state prior to checking for differences")
	planCmd.Flags().StringVar(&planState, "state", "", "Path to read and save the Terraform state")

	stfCmd.AddCommand(planCmd)
}
