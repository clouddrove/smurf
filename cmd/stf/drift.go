package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var driftDir string

// driftCmd defines a subcommand that detects drift between state and infrastructure for Terraform.
var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Detect drift between state and infrastructure for Terraform",
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.DetectDrift(driftDir)
	},
	Example: `
	smurf stf drift
	smurf stf drift --dir=/path/to/terraform
	`,
}

func init() {
	driftCmd.Flags().StringVar(&driftDir, "dir", ".", "Specify the directory containing Terraform configuration")
	stfCmd.AddCommand(driftCmd)
}