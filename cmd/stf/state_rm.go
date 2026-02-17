package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var (
	stateRmDir    string
	stateRmBackup bool
)

// stateRmCmd represents the command to remove resources from Terraform state
var stateRmCmd = &cobra.Command{
	Use:          "state-rm [address...]",
	Short:        "Remove resources from the Terraform state",
	Long:         `Remove one or more resources from the Terraform state file. This command is useful for unmanaging resources without destroying them.`,
	SilenceUsage: true,
	Args:         cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := terraform.StateRm(stateRmDir, args, stateRmBackup, useAI)

		if err != nil {
			terraform.ErrorHandler(err)
			return err
		}

		return nil
	},
	Example: `
    # Remove a single resource from state
    smurf stf state-rm aws_instance.example

    # Remove multiple resources
    smurf stf state-rm aws_instance.example aws_vpc.main

    # Remove resource from specific module
    smurf stf state-rm module.vpc.aws_subnet.private

    # Remove all resources of a type (using wildcard)
    smurf stf state-rm 'aws_instance.*'

    # Remove resource in a specific directory
    smurf stf state-rm --dir=path/to/terraform/code aws_instance.example

    # Remove without creating a backup
    smurf stf state-rm --no-backup aws_instance.example
    `,
}

func init() {
	stateRmCmd.Flags().StringVar(&stateRmDir, "dir", ".", "Specify the Terraform directory")
	stateRmCmd.Flags().BoolVar(&stateRmBackup, "backup", true, "Create a backup of the state file before removal")
	stateRmCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	stfCmd.AddCommand(stateRmCmd)
}
