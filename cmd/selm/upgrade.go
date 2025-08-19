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

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		if configs.Debug {
			pterm.EnableDebugMessages()
			pterm.Debug.Printfln("Starting Helm upgrade with configuration:")
			pterm.Debug.Printfln("  Release: %s", releaseName)
			pterm.Debug.Printfln("  Chart: %s", chartPath)
			pterm.Debug.Printfln("  Namespace: %s", configs.Namespace)
			pterm.Debug.Printfln("  Timeout: %v", timeoutDuration)
			pterm.Debug.Printfln("  Values files: %v", configs.File)
			pterm.Debug.Printfln("  Set values: %v", configs.Set)
			pterm.Debug.Printfln("  Set literal values: %v", configs.SetLiteral)
			pterm.Debug.Printfln("  Repo URL: %s", RepoURL)
			pterm.Debug.Printfln("  Version: %s", Version)
			pterm.Debug.Printfln("  Create namespace: %v", createNamespace)
			pterm.Debug.Printfln("  Install if not present: %v", installIfNotPresent)
			pterm.Debug.Printfln("  Atomic: %v", configs.Atomic)
		}

		// Check if release exists and handle install if not present
		if installIfNotPresent {
			exists, err := helm.HelmReleaseExists(releaseName, configs.Namespace)
			if err != nil {
				if configs.Debug {
					pterm.Debug.Printfln("Error checking release existence: %v", err)
				}
				return err
			}
			if !exists {
				if configs.Debug {
					pterm.Debug.Printfln("Release not found, performing installation...")
				}
				if err := helm.HelmInstall(releaseName, chartPath, configs.Namespace, configs.File, timeoutDuration, configs.Atomic, configs.Debug, configs.Set, configs.SetLiteral, RepoURL, Version); err != nil {
					return err
				}
				pterm.Success.Printfln("Release '%s' installed successfully", releaseName)
				return nil
			}
			if configs.Debug {
				pterm.Debug.Printfln("Release exists, proceeding with upgrade")
			}
		}

		err := helm.HelmUpgrade(
			releaseName,
			chartPath,
			configs.Namespace,
			configs.Set,
			configs.File,
			configs.SetLiteral,
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

		pterm.Success.Printfln("Release '%s' upgraded successfully", releaseName)
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
	upgradeCmd.Flags().StringSliceVar(&configs.Set, "set", []string{}, "Set values on the command line")
	upgradeCmd.Flags().StringSliceVar(&configs.SetLiteral, "set-literal", []string{}, "Set literal values on the command line")
	upgradeCmd.Flags().StringSliceVarP(&configs.File, "values", "f", []string{}, "Specify values in a YAML file")
	upgradeCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "default", "Specify the namespace")
	upgradeCmd.Flags().BoolVar(&createNamespace, "create-namespace", false, "Create the namespace if it does not exist")
	upgradeCmd.Flags().BoolVar(&configs.Atomic, "atomic", false, "Roll back changes on failure")
	upgradeCmd.Flags().IntVar(&configs.Timeout, "timeout", 150, "Timeout for Kubernetes operations")
	upgradeCmd.Flags().BoolVar(&configs.Debug, "debug", false, "Enable verbose output")
	upgradeCmd.Flags().BoolVar(&installIfNotPresent, "install", false, "Install if not present")
	upgradeCmd.Flags().StringVar(&RepoURL, "repo-url", "", "Helm repository URL")
	upgradeCmd.Flags().StringVar(&Version, "version", "", "Helm chart version")
	selmCmd.AddCommand(upgradeCmd)
}
