package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var varNameValue string
var varFile string

// planCmd defines a subcommand that generates and shows an execution plan for Terraform
var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate and show an execution plan for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Plan(varNameValue, varFile)
	},
	Example: `
	smurf stf plan
	`,
}

func init() {
	// Add flags for -var and -var-file
	planCmd.Flags().StringVar(&varNameValue, "var", "", "Specify a variable in 'NAME=VALUE' format")
	planCmd.Flags().StringVar(&varFile, "var-file", "", "Specify a file containing variables")

	stfCmd.AddCommand(planCmd)
}
