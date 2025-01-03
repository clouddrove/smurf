package stf

import (
	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// applyCmd defines a subcommand that applies the changes required to reach the desired state of Terraform Infrastructure.
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the changes required to reach the desired state of Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {
		approve := configs.CanApply
		return terraform.Apply(approve)
	},
	Example: `
	smurf stf apply
	`,
}

func init() {
	stfCmd.Flags().BoolVar(&configs.CanApply, "approve", true, "Skip interactive approval of plan before applying")
	stfCmd.AddCommand(applyCmd)
}
