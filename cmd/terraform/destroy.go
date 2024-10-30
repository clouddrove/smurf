package terraform

import (
	"github.com/clouddrove/smurf/cmd"
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy the Terraform Infrastructure",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Destroy()
	},
}

func init() {
	cmd.RootCmd.AddCommand(destroyCmd)
}
