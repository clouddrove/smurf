package helm

import (
	"fmt"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Helm releases.",
	RunE: func(cmd *cobra.Command, args []string) error {
		releases, err := helm.HelmList("default") 
		if err != nil {
			return err
		}
		for _, release := range releases {
			fmt.Printf("%v\n", release.Name)
		}
		return nil
	},
	Example: `
	smurf selm list
	`,
}

func init() {
	selmCmd.AddCommand(listCmd)
}
