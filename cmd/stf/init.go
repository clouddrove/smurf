package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var initUpgrade bool

// initCmd defines a subcommand that initializes Terraform.
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {

		return terraform.Init(initUpgrade)
	},
	Example: `
	smurf stf init
	`,
}

func init() {
	initCmd.Flags().BoolVar(&initUpgrade, "upgrade", true, "Upgrade the Terraform modules and plugins")
	stfCmd.AddCommand(initCmd)
}
