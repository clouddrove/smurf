package stf

import (
    "github.com/clouddrove/smurf/internal/terraform"
    "github.com/spf13/cobra"
)

var planVarNameValue []string
var planVarFile []string
var planDir string

// planCmd defines a subcommand that generates and shows an execution plan for Terraform
var planCmd = &cobra.Command{
    Use:   "plan",
    Short: "Generate and show an execution plan for Terraform",
    RunE: func(cmd *cobra.Command, args []string) error {
        return terraform.Plan(planVarNameValue, planVarFile, planDir)
    },
    Example: `
    smurf stf plan

    # Specify variables
    smurf stf plan -var="region=us-west-2"

    # Specify multiple variables
    smurf stf plan -var="region=us-west-2" -var="instance_type=t2.micro"

    # Specify a custom directory
    smurf stf plan --dir=/path/to/terraform/files
    `,
}

func init() {
    planCmd.Flags().StringArrayVar(&planVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
    planCmd.Flags().StringArrayVar(&planVarFile, "var-file", []string{}, "Specify a file containing variables")
    planCmd.Flags().StringVar(&planDir, "dir", ".", "Specify the directory containing Terraform files")

    stfCmd.AddCommand(planCmd)
}
