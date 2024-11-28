package terraform

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the changes required to reach the desired state of Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Apply()
	},
	Example: `
	smurf stf apply
	`,
}

func init() {
	stfCmd.AddCommand(applyCmd)
}
