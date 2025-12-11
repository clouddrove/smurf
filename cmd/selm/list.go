package selm

import (
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	allNamespaces bool
	outputFormat  string
	namespace     string
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List Helm releases",
	Long: `List Helm releases across namespaces with various output formats.
Defaults to showing releases in the default namespace unless specified.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if allNamespaces {
			namespace = "" // Empty namespace means all namespaces
		}
		if cmd.Flags().Changed("namespace") && allNamespaces {
			pterm.Warning.Println("--namespace is ignored when --all-namespaces is specified")
		}

		_, err := helm.ListReleases(namespace, outputFormat, useAI)
		return err
	},
	Example: `
  # List in current namespace (default)
  smurf selm list

  # List in specific namespace
  smurf selm list -n mynamespace

  # List across all namespaces
  smurf selm list -A

  # List with JSON output 
  smurf selm list -n kube-system -o json

  # List with YAML output (short flags)
  smurf selm ls -A -o yaml
`,
}

func init() {
	listCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "list across all namespaces")
	listCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace scope for listing")
	listCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table|json|yaml)")
	listCmd.Flags().BoolVar(&useAI, "ai", false, "Enable AI help mode")

	// Register completion functions
	listCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveDefault
	})

	selmCmd.AddCommand(listCmd)
}
