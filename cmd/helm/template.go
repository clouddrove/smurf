package helm

import (
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)


var templateCmd = &cobra.Command{
    Use:   "template [RELEASE] [CHART]",
    Short: "Render chart templates ",
    Args:  cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        return helm.HelmTemplate(args[0], args[1], "default")
    },
}

func init() {
    selmCmd.AddCommand(templateCmd)
}
