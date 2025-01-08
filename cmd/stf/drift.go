package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// driftCmd defines a subcommand that detects drift between state and infrastructure for Terraform.
var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between state and infrastructure  for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.DetectDrift()
	},
	Example: `
	smurf stf drift
	`,
}

func init() {
	stfCmd.AddCommand(driftCmd)
}
