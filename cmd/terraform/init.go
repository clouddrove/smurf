package terraform

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Init()
	},
	Example: `
	smurf stf init
	`,
}

func init() {
	stfCmd.AddCommand(initCmd)
}
