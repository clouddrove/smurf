package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var provisionApprove bool // deprecated: use provisionAutoApprove via --auto-approve
var provisionAutoApprove bool
var varNameValue []string
var varFile []string
var lock bool
var upgrade bool
var provisionDir string

var provisionCmd = &cobra.Command{
	Use:          "provision",
	Short:        "Its the combination of init, plan, apply, output for Terraform",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --approve is a deprecated alias for --auto-approve; honor either one.
		autoApprove := provisionAutoApprove || provisionApprove

		if err := terraform.Init(provisionDir, upgrade, useAI); err != nil {
			return err
		}

		if _, err := terraform.Plan(varNameValue, varFile, provisionDir, planDestroy, planTarget, planRefresh, planState, planOut, useAI); err != nil {
			return err
		}

		if err := terraform.Apply(autoApprove, varNameValue, varFile, lock, provisionDir, applyTarget, applyState, useAI); err != nil {
			return err
		}

		if err := terraform.Output(provisionDir, "table", useAI); err != nil {
			return err
		}

		return nil
	},
	Example: `
	# Prompts for interactive approval before applying
	smurf stf provision

	# Skip interactive approval of plan before applying
	smurf stf provision --dir=/path/to/terraform/files --auto-approve
	`,
}

func init() {
	provisionCmd.Flags().StringArrayVar(&varNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	provisionCmd.Flags().StringArrayVar(&varFile, "var-file", []string{}, "Specify a file containing variables")
	provisionCmd.Flags().BoolVar(&provisionAutoApprove, "auto-approve", false, "Skip interactive approval of plan before applying")
	provisionCmd.Flags().BoolVar(&provisionApprove, "approve", false, "Skip interactive approval of plan before applying")
	_ = provisionCmd.Flags().MarkDeprecated("approve", "use --auto-approve instead")
	provisionCmd.Flags().BoolVar(&lock, "lock", true, "Hold a state lock during the operation (disable with --lock=false)")
	provisionCmd.Flags().BoolVar(&upgrade, "upgrade", false, "Upgrade the Terraform modules and plugins to the latest versions")
	provisionCmd.Flags().StringVar(&provisionDir, "dir", "", "Specify the directory for Terraform operations")
	provisionCmd.Flags().StringVar(&planOut, "out", "", "Path to save the generated execution plan")
	provisionCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")
	stfCmd.AddCommand(provisionCmd)
}
