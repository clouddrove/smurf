package helm

import (
    "github.com/clouddrove/smurf/internal/helm"
    "github.com/spf13/cobra"
    "time"
)

var (
    setValues       []string
    valuesFiles     []string
    namespace       string
    createNamespace bool
    atomic          bool
    timeout         time.Duration
    debug           bool
    installIfNotPresent bool 
)

var upgradeCmd = &cobra.Command{
    Use:   "upgrade [NAME] [CHART]",
    Short: "Upgrade a deployed Helm chart.",
    Args:  cobra.ExactArgs(2),
    RunE: func(cmd *cobra.Command, args []string) error {
        releaseName := args[0]
        chartPath := args[1]
        if installIfNotPresent {
            exists, err := helm.HelmReleaseExists(releaseName, namespace)
            if err != nil {
                return err 
            }
            if !exists {
                if err := helm.HelmInstall(releaseName, chartPath, namespace); err != nil {
                    return err
                }
            }
        }
        return helm.HelmUpgrade(releaseName, chartPath, namespace, setValues, valuesFiles, createNamespace, atomic, timeout, debug)
    },
}

func init() {
    selmCmd.AddCommand(upgradeCmd)
    upgradeCmd.Flags().StringSliceVar(&setValues, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
    upgradeCmd.Flags().StringSliceVarP(&valuesFiles, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")
    upgradeCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Specify the namespace to install the release into")
    upgradeCmd.Flags().BoolVar(&createNamespace, "create-namespace", false, "Create the namespace if it does not exist")
    upgradeCmd.Flags().BoolVar(&atomic, "atomic", false, "If set, the installation process purges the chart on fail, the upgrade process rolls back changes, and the upgrade process waits for the resources to be ready")
    upgradeCmd.Flags().DurationVar(&timeout, "timeout", 300*time.Second, "Time to wait for any individual Kubernetes operation (like Jobs for hooks)")
    upgradeCmd.Flags().BoolVar(&debug, "debug", false, "Enable verbose output")
    upgradeCmd.Flags().BoolVar(&installIfNotPresent, "install", false, "Install the chart if it is not already installed")
}
