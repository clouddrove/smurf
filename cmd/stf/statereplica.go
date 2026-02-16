package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// stateReplaceProviderCmd replaces provider in the Terraform state
var stateReplaceProviderCmd = &cobra.Command{
	Use:   "replace-provider FROM_PROVIDER TO_PROVIDER",
	Short: "Replace provider in the Terraform state",
	Long: `Replace provider configuration in the Terraform state.
This is useful when migrating from one provider source to another.`,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return terraform.StateReplaceProvider(stateDir, args[0], args[1], useAI)
	},
	Example: `
    # Replace provider source
    smurf stf state replace-provider "registry.terraform.io/-/aws" "hashicorp/aws"
    
    # From custom directory
    smurf stf state replace-provider --dir=/path/to/terraform "registry.terraform.io/-/aws" "hashicorp/aws"
    `,
}

func init() {
	stateCmd.AddCommand(stateReplaceProviderCmd)
}
