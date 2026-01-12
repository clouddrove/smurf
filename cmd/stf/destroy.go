package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var destroyApprove bool
var destroyAutoApprove bool
var destroyLock bool
var destroyDir string
var destroyVarNameValue []string
var destroyVarFile []string

// destroyCmd defines a subcommand that destroys the Terraform Infrastructure.
var destroyCmd = &cobra.Command{
	Use:          "destroy",
	Short:        "Destroy the Terraform Infrastructure",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use either approve or auto-approve flag
		approve := destroyApprove || destroyAutoApprove
		err := terraform.Destroy(approve, destroyLock, destroyDir, destroyVarNameValue, destroyVarFile, useAI) // UPDATED: added new parameters
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
	smurf stf destroy
	# Skip approval prompt
	smurf stf destroy --auto-approve
	# Specify a custom directory
	smurf stf destroy --dir=/path/to/terraform
	# NEW: Use variable files
	smurf stf destroy --var-file=production.tfvars
	smurf stf destroy --var-file=common.tfvars --var-file=production.tfvars
	# NEW: Use variables
	smurf stf destroy --var="environment=staging"
	# Combined usage
	smurf stf destroy --auto-approve --var-file=prod.tfvars --var="force_destroy=true"
`,
}

func init() {
	destroyCmd.Flags().BoolVar(&destroyApprove, "approve", false, "Skip interactive approval of plan before applying")
	destroyCmd.Flags().BoolVar(&destroyAutoApprove, "auto-approve", false, "Skip interactive approval of plan before destroying")
	destroyCmd.Flags().BoolVar(&destroyLock, "lock", true, "Don't hold a state lock during the operation. This is dangerous if others might concurrently run commands against the same workspace.")
	destroyCmd.Flags().StringVar(&destroyDir, "dir", ".", "Specify the directory containing Terraform configuration")

	// NEW: Add var and var-file flags
	destroyCmd.Flags().StringArrayVar(&destroyVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	destroyCmd.Flags().StringArrayVar(&destroyVarFile, "var-file", []string{}, "Specify a file containing variables")
	destroyCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	stfCmd.AddCommand(destroyCmd)
}
