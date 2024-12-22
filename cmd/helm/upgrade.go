package helm

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
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
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade [NAME] [CHART]",
	Short: "Upgrade a deployed Helm chart.",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var releaseName, chartPath string

		if len(args) >= 1 {
			releaseName = args[0]
		}
		if len(args) >= 2 {
			chartPath = args[1]
		}

		if releaseName == "" || chartPath == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if releaseName == "" {
				releaseName = data.Selm.ReleaseName
				if releaseName == "" {
					releaseName = filepath.Base(data.Selm.ChartName)
				}
			}
			if chartPath == "" {
				chartPath = data.Selm.ChartName
			}

			if releaseName == "" || chartPath == "" {
				return errors.New(color.RedString("RELEASE and CHART must be provided either as arguments or in the config"))
			}

			if namespace == "default" && data.Selm.Namespace != "" {
				namespace = data.Selm.Namespace
			}
		}

		if releaseName == "" || chartPath == "" {
			return errors.New(color.RedString("RELEASE and CHART must be provided"))
		}

		timeoutDuration := time.Duration(timeout) * time.Second

		if installIfNotPresent {
			exists, err := helm.HelmReleaseExists(releaseName, namespace)
			if err != nil {
				return fmt.Errorf("failed to check if Helm release exists: %w", err)
			}
			if !exists {
				if err := helm.HelmInstall(releaseName, chartPath, namespace, valuesFiles, timeoutDuration, atomic, debug, setValues); err != nil {
					return fmt.Errorf(color.RedString("Helm install failed: %v", err))
				}
			}
		}

		if namespace == "" {
			namespace = "default"
		}

		err := helm.HelmUpgrade(
			releaseName,
			chartPath,
			namespace,
			setValues,
			valuesFiles,
			createNamespace,
			atomic,
			timeoutDuration,
			debug,
		)
		if err != nil {
			return fmt.Errorf(color.RedString("Helm upgrade failed: %v", err))
		}
		pterm.Success.Println("Helm chart upgraded successfully.")
		return nil
	},
	Example: `
smurf selm upgrade my-release ./mychart
smurf selm upgrade my-release ./mychart -n my-namespace
smurf selm upgrade my-release ./mychart --set key1=val1,key2=val2
smurf selm upgrade my-release ./mychart -f values.yaml --timeout 600 --atomic --debug --install
smurf selm upgrade
# In the last example, it will read RELEASE and CHART from the config file
`,
}

func init() {
	upgradeCmd.Flags().StringSliceVar(&setValues, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	upgradeCmd.Flags().StringSliceVarP(&valuesFiles, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")
	upgradeCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Specify the namespace to install the release into")
	upgradeCmd.Flags().BoolVar(&createNamespace, "create-namespace", false, "Create the namespace if it does not exist")
	upgradeCmd.Flags().BoolVar(&atomic, "atomic", false, "If set, the installation process purges the chart on fail, the upgrade process rolls back changes, and the upgrade process waits for the resources to be ready")
	upgradeCmd.Flags().IntVar(&timeout, "timeout", 150, "Time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	upgradeCmd.Flags().BoolVar(&debug, "debug", false, "Enable verbose output")
	upgradeCmd.Flags().BoolVar(&installIfNotPresent, "install", false, "Install the chart if it is not already installed")
	selmCmd.AddCommand(upgradeCmd)
}
