package terraform

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between state and infrastructure  for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.DetectDrift()
	},
}

func init() {
	stfCmd.AddCommand(driftCmd)
}
