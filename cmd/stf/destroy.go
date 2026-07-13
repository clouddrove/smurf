package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var destroyApprove bool // deprecated: use destroyAutoApprove via --auto-approve
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
		// --approve is a deprecated alias for --auto-approve; honor either one.
		approve := destroyAutoApprove || destroyApprove
		return terraform.Destroy(approve, destroyLock, destroyDir, destroyVarNameValue, destroyVarFile, useAI)
	},
	Example: `
	# simple smurf stf destroy commad
	smurf stf destroy

	# Skip approval prompt
	smurf stf destroy --auto-approve

	# Specify a custom directory
	smurf stf destroy --dir=/path/to/terraform

	# Use variable files
	smurf stf destroy --var-file=production.tfvars
	smurf stf destroy --var-file=common.tfvars --var-file=production.tfvars

	# Use variables
	smurf stf destroy --var="environment=staging"
	
	# Combined usage
	smurf stf destroy --auto-approve --var-file=prod.tfvars --var="force_destroy=true"
`,
}

func init() {
	destroyCmd.Flags().BoolVar(&destroyAutoApprove, "auto-approve", false, "Skip interactive approval of plan before destroying")
	destroyCmd.Flags().BoolVar(&destroyApprove, "approve", false, "Skip interactive approval of plan before applying")
	_ = destroyCmd.Flags().MarkDeprecated("approve", "use --auto-approve instead")
	destroyCmd.Flags().BoolVar(&destroyLock, "lock", true, "Hold a state lock during the operation (disable with --lock=false)")
	destroyCmd.Flags().StringVar(&destroyDir, "dir", ".", "Specify the directory containing Terraform configuration")

	// NEW: Add var and var-file flags
	destroyCmd.Flags().StringArrayVar(&destroyVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	destroyCmd.Flags().StringArrayVar(&destroyVarFile, "var-file", []string{}, "Specify a file containing variables")
	destroyCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	stfCmd.AddCommand(destroyCmd)
}
