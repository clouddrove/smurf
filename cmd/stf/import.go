package stf

import (
	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

var importAddr string
var importID string
var importDir string
var importVarNameValue []string
var importVarFile []string
var importRefresh bool
var importState string
var importConfig string
var importAllowMissing bool

// importCmd defines a subcommand that imports existing infrastructure into Terraform state
var importCmd = &cobra.Command{
	Use:          "import [flags] ADDRESS ID",
	Short:        "Import existing infrastructure into Terraform state",
	SilenceUsage: true,
	Args:         cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		importAddr = args[0]
		importID = args[1]
		// Note: target flag is intentionally omitted as it's not supported by import
		return terraform.Import(importAddr, importID, importDir, importVarNameValue, importVarFile,
			[]string{}, importRefresh, importState, importConfig, importAllowMissing, useAI)
	},
	Example: `
    # Import a basic resource
    smurf stf import aws_instance.web i-1234567890abcdef0

    # Import a resource in a module
    smurf stf import module.vpc.aws_vpc.main vpc-12345678

    # Import with custom directory
    smurf stf import --dir=/path/to/terraform/files aws_instance.web i-1234567890abcdef0

    # Import with variables
    smurf stf import -var="region=us-west-2" aws_instance.web i-1234567890abcdef0

    # Import with variable file
    smurf stf import --var-file=vars.tfvars aws_instance.web i-1234567890abcdef0

    # Import with custom state file
    smurf stf import --state=prod.tfstate aws_instance.web i-1234567890abcdef0

    # Import with config file
    smurf stf import --config=alternate.tf aws_instance.web i-1234567890abcdef0

    # Import multiple resources
    smurf stf import aws_instance.web i-1234567890abcdef0
    smurf stf import aws_security_group.web sg-12345678

    # Import without refresh (handled automatically by Terraform)
    smurf stf import --refresh=false aws_instance.web i-1234567890abcdef0

    # Allow missing resource during import
    smurf stf import --allow-missing aws_instance.web i-1234567890abcdef0

    # Complex example with multiple flags
    smurf stf import --dir=environments/prod --state=prod.tfstate --var-file=prod.tfvars \
      --allow-missing aws_instance.web i-1234567890abcdef0
    `,
}

func init() {
	importCmd.Flags().StringVar(&importDir, "dir", ".", "Specify the directory containing Terraform files")
	importCmd.Flags().StringArrayVar(&importVarNameValue, "var", []string{}, "Specify a variable in 'NAME=VALUE' format")
	importCmd.Flags().StringArrayVar(&importVarFile, "var-file", []string{}, "Specify a file containing variables")
	importCmd.Flags().BoolVar(&importRefresh, "refresh", true, "Update state prior to import (handled automatically by Terraform)")
	importCmd.Flags().StringVar(&importState, "state", "", "Path to read and save the Terraform state")
	importCmd.Flags().StringVar(&importConfig, "config", "", "Path to a Terraform configuration file to use for import")
	importCmd.Flags().BoolVar(&importAllowMissing, "allow-missing", false, "Allow import even if the configuration block is missing")
	importCmd.Flags().BoolVar(&useAI, "ai", false, "To enable AI help mode, export the OPENAI_API_KEY environment variable with your OpenAI API key.")

	stfCmd.AddCommand(importCmd)
}
