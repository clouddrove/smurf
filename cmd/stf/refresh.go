package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	refreshVars     []string
	refreshVarFiles []string
	refreshLock     bool
)

// refreshCmd represents the command to refresh the state of Terraform resources
var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Update the state file of your infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Refresh(refreshVars, refreshVarFiles, refreshLock)
	},
	Example: `
    # Basic refresh
    smurf stf refresh

    # Refresh with variables
    smurf stf refresh -var="region=us-west-2"

    # Refresh with variable file
    smurf stf refresh -var-file="prod.tfvars"

    # Refresh without state locking
    smurf stf refresh --lock=false
    `,
}

func init() {
	refreshCmd.Flags().StringArrayVar(&refreshVars, "var", []string{}, "Set a variable in 'name=value' format")
	refreshCmd.Flags().StringArrayVar(&refreshVarFiles, "var-file", []string{}, "Path to a Terraform variable file")
	refreshCmd.Flags().BoolVar(&refreshLock, "lock", true, "Lock the state file when running operation (defaults to true)")

	stfCmd.AddCommand(refreshCmd)
}
