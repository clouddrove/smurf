package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var outputDir string

// outputCmd defines a subcommand that generates output for the current state of Terraform Infrastructure.
var outputCmd = &cobra.Command{
	Use:           "output",
	Short:         "Generate output for the current state of Terraform Infrastructure",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.Output(outputDir)
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
	smurf stf output
	smurf stf output --dir <terraform-directory>
	`,
}

func init() {
	outputCmd.Flags().StringVar(&outputDir, "dir", ".", "Specify the Terraform directory")
	stfCmd.AddCommand(outputCmd)
}
