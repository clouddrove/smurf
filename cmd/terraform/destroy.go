package terraform

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy the Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Destroy()
	},
	Example: `
	smurf stf destroy
	`,
}

func init() {
	stfCmd.AddCommand(destroyCmd)
}
