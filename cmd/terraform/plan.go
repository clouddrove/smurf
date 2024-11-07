package terraform

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var varNameValue string
var varFile string

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate and show an execution plan for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Plan(varNameValue, varFile)
	},
}

func init() {
	// Add flags for -var and -var-file
	planCmd.Flags().StringVar(&varNameValue, "var", "", "Specify a variable in 'NAME=VALUE' format")
	planCmd.Flags().StringVar(&varFile, "var-file", "", "Specify a file containing variables")

	stfCmd.AddCommand(planCmd)
}
