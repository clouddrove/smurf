package terraform

import (
	"github.com/clouddrove/smurf/cmd"
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate and show an execution plan for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Plan()
	},
}

func init() {
	cmd.RootCmd.AddCommand(planCmd)
}
