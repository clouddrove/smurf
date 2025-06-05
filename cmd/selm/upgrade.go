package selm

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	createNamespace     bool
	installIfNotPresent bool
)

// upgradeCmd facilitates upgrading an existing Helm release or installing it if it's not present
// (depending on the `--install` flag). It supports specifying a custom namespace, waiting for
// resources to become ready (`--atomic`), setting arbitrary values (`--set` and `--values`),
// and other typical Helm upgrade options. If no arguments are provided, it attempts to read
// settings from the config file.
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
				return err
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
				return errors.New("RELEASE and CHART must be provided either as arguments or in the config")
			}

			if configs.Namespace == "default" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if releaseName == "" || chartPath == "" {
			pterm.Error.Printfln("RELEASE and CHART must be provided")
			return errors.New("RELEASE and CHART must be provided")
		}

		timeoutDuration := time.Duration(configs.Timeout) * time.Second

		if installIfNotPresent {
			exists, err := helm.HelmReleaseExists(releaseName, configs.Namespace)
			if err != nil {
				return err
			}
			if !exists {
				if err := helm.HelmInstall(releaseName, chartPath, configs.Namespace, configs.File, timeoutDuration, configs.Atomic, configs.Debug, configs.Set, []string{}, RepoURL, Version); err != nil {
					return err
				}
			}
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		err := helm.HelmUpgrade(
			releaseName,
			chartPath,
			configs.Namespace,
			configs.Set,
			configs.File,
			[]string{},
			createNamespace,
			configs.Atomic,
			timeoutDuration,
			configs.Debug,
			RepoURL,
			Version,
		)
		if err != nil {
			return err
		}

		pterm.Success.Println("Helm chart upgraded successfully.")
		return nil
	},
	Example: `
smurf selm upgrade my-release ./mychart
smurf selm upgrade my-release ./mychart -n my-namespace
smurf selm upgrade my-release ./mychart --set key1=val1,key2=val2
smurf selm upgrade my-release ./mychart -f values.yaml --timeout 600 --atomic --debug --install
smurf selm upgrade my-release ./mychart --repo-url https://charts.example.com --version 1.2.3
smurf selm upgrade my-release ./mychart --set key1=val1 --set key2=val2
smurf selm upgrade my-release ./mychart --set-literal myPassword='MySecurePass!'
smurf selm upgrade
# In the last example, it will read RELEASE and CHART from the config file
`,
}

func init() {
	upgradeCmd.Flags().StringSliceVar(&configs.Set, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	upgradeCmd.Flags().StringSliceVar(&configs.SetLiteral, "set-literal", []string{}, "Set literal values on the command line (values are always treated as strings)")
	upgradeCmd.Flags().StringSliceVarP(&configs.File, "values", "f", []string{}, "Specify values in a YAML file (can specify multiple)")
	upgradeCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "default", "Specify the namespace to install the release into")
	upgradeCmd.Flags().BoolVar(&createNamespace, "create-namespace", false, "Create the namespace if it does not exist")
	upgradeCmd.Flags().BoolVar(&configs.Atomic, "atomic", false, "If set, the installation process purges the chart on fail, the upgrade process rolls back changes, and the upgrade process waits for the resources to be ready")
	upgradeCmd.Flags().IntVar(&configs.Timeout, "timeout", 150, "Time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	upgradeCmd.Flags().BoolVar(&configs.Debug, "debug", false, "Enable verbose output")
	upgradeCmd.Flags().BoolVar(&installIfNotPresent, "install", false, "Install the chart if it is not already installed")
	upgradeCmd.Flags().StringVar(&RepoURL, "repo-url", "", "Helm repository URL")
	upgradeCmd.Flags().StringVar(&Version, "version", "", "Helm chart version")
	selmCmd.AddCommand(upgradeCmd)
}
