package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var approve bool

// applyCmd defines a subcommand that applies the changes required to reach the desired state of Terraform Infrastructure.
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply the changes required to reach the desired state of Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.Apply(approve)
	},
	Example: `
	smurf stf apply
	`,
}

func init() {
	applyCmd.Flags().BoolVar(&approve, "approve", false, "Skip interactive approval of plan before applying")
	stfCmd.AddCommand(applyCmd)
}
