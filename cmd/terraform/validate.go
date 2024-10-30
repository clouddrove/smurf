package terraform

import (
	"github.com/clouddrove/smurf/cmd"
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate  Terraform changes",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Validate()
	},
}

func init() {
	cmd.RootCmd.AddCommand(validateCmd)
}
