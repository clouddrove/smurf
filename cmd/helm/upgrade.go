package helm

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	setValues           []string
	valuesFiles         []string
	namespace           string
	createNamespace     bool
	atomic              bool
	timeout             int
	debug               bool
	installIfNotPresent bool
	autoUpgrade         bool
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [NAME] [CHART]",
	Short: "Upgrade a deployed Helm chart.",
	RunE: func(cmd *cobra.Command, args []string) error {
		timeoutDuration := time.Duration(timeout) * time.Second
		if autoUpgrade {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if data.Selm.ChartName == "" {
				return fmt.Errorf("chart name is required in config")
			}

			releaseName := data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			currentNamespace := namespace
			if currentNamespace == "" {
				currentNamespace = data.Selm.Namespace
			}
			if currentNamespace == "" {
				currentNamespace = "default"
			}

			if len(args) < 2 {
				args = []string{releaseName, data.Selm.ChartName}
			}

			if installIfNotPresent {
				exists, err := helm.HelmReleaseExists(args[0], currentNamespace)
				if err != nil {
					return err
				}

				if !exists {

					if err := helm.HelmInstall(args[0], args[1], currentNamespace, valuesFiles, timeoutDuration, atomic, debug, setValues); err != nil {
						return err
					}
				}
			}
			
			return helm.HelmUpgrade(
				args[0],
				args[1],
				currentNamespace,
				setValues,
				valuesFiles,
				createNamespace,
				atomic,
				timeoutDuration,
				debug,
			)
		}

		releaseName := args[0]
		chartPath := args[1]
		if installIfNotPresent {
			exists, err := helm.HelmReleaseExists(releaseName, namespace)
			if err != nil {
				return err
			}
			if !exists {
				if err := helm.HelmInstall(args[0], args[1], namespace, valuesFiles, timeoutDuration, atomic, debug, setValues); err != nil {
					return err
				}
			}
		}
		return helm.HelmUpgrade(releaseName, chartPath, namespace, setValues, valuesFiles, createNamespace, atomic, timeoutDuration, debug, )
	},
	Example: `
    smurf selm upgrade my-release ./mychart
    smurf selm upgrade my-release ./mychart -n my-namespace
    smurf selm upgrade my-release ./mychart --set key1=val1,key2=val2
    smurf selm upgrade my-release ./mychart -f values.yaml --timeout 600 --atomic --debug --install
    `,
}

func init() {
	upgradeCmd.Flags().StringSliceVar(&setValues, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	upgradeCmd.Flags().StringSliceVarP(&valuesFiles, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")
	upgradeCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Specify the namespace to install the release into")
	upgradeCmd.Flags().BoolVar(&createNamespace, "create-namespace", false, "Create the namespace if it does not exist")
	upgradeCmd.Flags().BoolVar(&atomic, "atomic", false, "If set, the installation process purges the chart on fail, the upgrade process rolls back changes, and the upgrade process waits for the resources to be ready")
	upgradeCmd.Flags().IntVar(&timeout, "timeout", 300, "Time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	upgradeCmd.Flags().BoolVar(&debug, "debug", false, "Enable verbose output")
	upgradeCmd.Flags().BoolVar(&installIfNotPresent, "install", false, "Install the chart if it is not already installed")
	upgradeCmd.Flags().BoolVar(&autoUpgrade, "auto", false, "Upgrade Helm chart automatically")
	selmCmd.AddCommand(upgradeCmd)
}
