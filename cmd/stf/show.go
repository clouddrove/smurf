package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var showVarNameValue []string
var showVarFile []string
var showDir string
var showPlanFile string
var showJSON bool
var showResource string

// showCmd defines a subcommand that displays Terraform state or plan details
var showCmd = &cobra.Command{
	Use:          "show [plan-file]",
	Short:        "Show Terraform state or saved plan details",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If a plan file is provided as positional argument
		planFile := showPlanFile
		if len(args) > 0 && showPlanFile == "" {
			planFile = args[0]
		}

		// Show plan if plan file is specified
		if planFile != "" {
			return terraform.ShowPlan(planFile, showVarNameValue, showVarFile, showDir, showJSON, useAI)
		}

		// Show resource if specified
		if showResource != "" {
			return terraform.ShowResource(showResource, showVarNameValue, showVarFile, showDir, showJSON, useAI)
		}

		// Default: show current state
		return terraform.ShowState(showVarNameValue, showVarFile, showDir, showJSON, useAI)
	},
	Example: `
    # Show current state
    smurf stf show

    # Show state in JSON format
    smurf stf show --json

    # Show saved plan file
    smurf stf show plan.out
    smurf stf show --plan=plan.out

    # Show plan file in JSON format
    smurf stf show plan.out --json

    # Show specific resource from state
    smurf stf show --resource=aws_instance.web
    smurf stf show --resource=module.vpc

    # Show resource in JSON format
    smurf stf show --resource=aws_instance.web --json

    # Show state from custom directory
    smurf stf show --dir=environments/prod

    # Show state with variables
    smurf stf show --var="region=us-west-2"
    smurf stf show --var-file=vars.tfvars
    `,
}

func init() {
	showCmd.Flags().StringArrayVar(&showVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	showCmd.Flags().StringArrayVar(&showVarFile, "var-file", []string{}, "Specify a file containing variables")
	showCmd.Flags().StringVar(&showDir, "dir", ".", "Specify the directory containing Terraform files")
	showCmd.Flags().StringVar(&showPlanFile, "plan", "", "Path to a saved plan file to show")
	showCmd.Flags().BoolVar(&showJSON, "json", false, "Output in JSON format")
	showCmd.Flags().StringVar(&showResource, "resource", "", "Show specific resource by address (e.g., aws_instance.web)")
	showCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	stfCmd.AddCommand(showCmd)
}
