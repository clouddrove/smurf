package stf

import (
    "github.com/clouddrove/smurf/internal/terraform"
    "github.com/spf13/cobra"
)

var (
    initUpgrade bool
    initDir     string  // New variable for directory flag
)

// initCmd defines a subcommand that initializes Terraform.
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize Terraform",
    RunE: func(cmd *cobra.Command, args []string) error {
        return terraform.Init(initDir, initUpgrade)
    },
    Example: `
  smurf stf init
  smurf stf init --dir=/path/to/terraform/files
`,
}

func init() {
    initCmd.Flags().BoolVar(&initUpgrade, "upgrade", true, "Upgrade the Terraform modules and plugins")
    initCmd.Flags().StringVar(&initDir, "dir", "", "Directory containing Terraform files (default is current directory)")  // Add directory flag
    stfCmd.AddCommand(initCmd)
}

