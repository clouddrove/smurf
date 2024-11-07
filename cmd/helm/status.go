package helm

import (
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var statusNamespace string

var statusCmd = &cobra.Command{
	Use:   "status [NAME]",
	Short: "Status of a Helm release.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		releaseName := args[0]
		if statusNamespace == "" { 
            uninstallNamespace = "default"
        }
		return helm.HelmStatus(releaseName, statusNamespace) 
	},
}

func init() {
	statusCmd.Flags().StringVarP(&statusNamespace, "namespace", "n", "", "Specify the namespace to get status of the Helm chart ")
	selmCmd.AddCommand(statusCmd)
}
