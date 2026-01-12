package stf

import (
	"os"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	initUpgrade bool
	initDir     string // New variable for directory flag
)

// initCmd defines a subcommand that initializes Terraform.
var initCmd = &cobra.Command{
	Use:           "init",
	Short:         "Initialize Terraform",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.Init(initDir, initUpgrade, useAI)
		if err != nil {
			os.Exit(1)
		}
		return nil
	},
	Example: `
  smurf stf init
  smurf stf init --dir=/path/to/terraform/files
`,
}

func init() {
	initCmd.Flags().BoolVar(&initUpgrade, "upgrade", true, "Upgrade the Terraform modules and plugins")
	initCmd.Flags().StringVar(&initDir, "dir", "", "Directory containing Terraform files (default is current directory)") // Add directory flag
	initCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	stfCmd.AddCommand(initCmd)
}
