package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var driftDir string

// driftCmd defines a subcommand that detects drift between state and infrastructure for Terraform.
var driftCmd = &cobra.Command{
	Use:          "drift",
	Short:        "Detect drift between state and infrastructure for Terraform",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.DetectDrift(driftDir, useAI)
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
	smurf stf drift
	smurf stf drift --dir=/path/to/terraform
	`,
}

func init() {
	driftCmd.Flags().StringVar(&driftDir, "dir", ".", "Specify the directory containing Terraform configuration")
	driftCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	stfCmd.AddCommand(driftCmd)
}
