package stf

import (
	"fmt"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/clouddrove/smurf/internal/utils"
	"github.com/spf13/cobra"
)

var (
	outputDir    string
	outputFormat string
)

// outputCmd defines a subcommand that generates output for the current state of Terraform Infrastructure.
var outputCmd = &cobra.Command{
	Use:          "output",
	Short:        "Generate output for the current state of Terraform Infrastructure",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !utils.ValidOutputFormat(outputFormat, "table", "json") {
			return fmt.Errorf("invalid output format %q: must be one of table, json", outputFormat)
		}
		return terraform.Output(outputDir, outputFormat, useAI)
	},
	Example: `
	smurf stf output
	smurf stf output --dir <terraform-directory>
	smurf stf output -o json
	`,
}

func init() {
	outputCmd.Flags().StringVar(&outputDir, "dir", ".", "Specify the Terraform directory")
	outputCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json)")
	outputCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	_ = outputCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json"}, cobra.ShellCompDirectiveDefault
	})

	stfCmd.AddCommand(outputCmd)
}
