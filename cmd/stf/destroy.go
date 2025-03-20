package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var destroyApprove bool
var destroyAutoApprove bool // New variable for auto-approve flag
var destroyLock bool
var destroyDir string

// destroyCmd defines a subcommand that destroys the Terraform Infrastructure.
var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy the Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use either approve or auto-approve flag
		approve := destroyApprove || destroyAutoApprove
		return terraform.Destroy(approve, destroyLock, destroyDir)
	},
	Example: `
	smurf stf destroy
	# Skip approval prompt
	smurf stf destroy --auto-approve
	# Specify a custom directory
	smurf stf destroy --dir=/path/to/terraform
`,
}

func init() {
	destroyCmd.Flags().BoolVar(&destroyApprove, "approve", false, "Skip interactive approval of plan before applying")
	destroyCmd.Flags().BoolVar(&destroyAutoApprove, "auto-approve", false, "Skip interactive approval of plan before destroying") // New flag
	destroyCmd.Flags().BoolVar(&destroyLock, "lock", true, "Don't hold a state lock during the operation. This is dangerous if others might concurrently run commands against the same workspace.")
	destroyCmd.Flags().StringVar(&destroyDir, "dir", ".", "Specify the directory containing Terraform configuration")
	stfCmd.AddCommand(destroyCmd)
}
