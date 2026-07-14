package stf

import (
	"fmt"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/spf13/cobra"
)

var (
	stateListDir    string
	stateListFormat string
)

// stateListCmd represents the command to list resources in the Terraform state
var stateListCmd = &cobra.Command{
	Use:          "state-list",
	Short:        "List resources in the Terraform state",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !utils.ValidOutputFormat(stateListFormat, "table", "json") {
			return fmt.Errorf("invalid output format %q: must be one of table, json", stateListFormat)
		}

		err := terraform.StateList(stateListDir, stateListFormat, useAI)
		if err != nil {
			if stateListFormat == "table" || stateListFormat == "" {
				terraform.ErrorHandler(err)
			}
			return err
		}

		return nil
	},
	Example: `
    # List all resources in state
    smurf stf state-list

    # List resources in a specific directory
    smurf stf state-list --dir=path/to/terraform/code

    # List resources as a JSON array
    smurf stf state-list -o json
    `,
}

func init() {
	stateListCmd.Flags().StringVar(&stateListDir, "dir", ".", "Specify the Terraform directory")
	stateListCmd.Flags().StringVarP(&stateListFormat, "output", "o", "table", "output format (table|json)")
	stateListCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	_ = stateListCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json"}, cobra.ShellCompDirectiveDefault
	})

	stfCmd.AddCommand(stateListCmd)
}
