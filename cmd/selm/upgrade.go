package selm

import (
	"errors"
	"fmt"
	"os"
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
	wait                bool
	historyMax          int
	useAI               bool
)

// upgradeCmd facilitates upgrading an existing Helm release or installing it if it's not present
var upgradeCmd = &cobra.Command{
	Use:          "upgrade [NAME] [CHART]",
	Short:        "Upgrade a deployed Helm chart.",
	Args:         cobra.MaximumNArgs(2),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if configs.Debug {
			pterm.EnableDebugMessages()
			pterm.Println("=== DEBUG MODE ENABLED ===")
		}

		var releaseName, chartPath string

		if len(args) >= 1 {
			releaseName = args[0]
			if configs.Debug {
				pterm.Printf("Release name from argument: %s\n", releaseName)
			}
		}
		if len(args) >= 2 {
			chartPath = args[1]
			if configs.Debug {
				pterm.Printf("Chart path from argument: %s\n", chartPath)
			}
		}

		if releaseName == "" || chartPath == "" {
			if configs.Debug {
				pterm.Println("Loading configuration from file...")
			}
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if releaseName == "" {
				releaseName = data.Selm.ReleaseName
				if releaseName == "" {
					releaseName = filepath.Base(data.Selm.ChartName)
				}
				if configs.Debug {
					pterm.Printf("Using release name from config: %s\n", releaseName)
				}
			}
			if chartPath == "" {
				chartPath = data.Selm.ChartName
				if configs.Debug {
					pterm.Printf("Using chart path from config: %s\n", chartPath)
				}
			}

			if releaseName == "" || chartPath == "" {
				return errors.New("RELEASE and CHART must be provided either as arguments or in the config")
			}

			if configs.Namespace == "default" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
				if configs.Debug {
					pterm.Printf("Using namespace from config: %s\n", configs.Namespace)
				}
			}
		}

		if releaseName == "" || chartPath == "" {
			pterm.Error.Printfln("RELEASE and CHART must be provided")
			return errors.New("RELEASE and CHART must be provided")
		}

		timeoutDuration := time.Duration(configs.Timeout) * time.Second

		if configs.Debug {
			pterm.Printf("Configuration\n")
			pterm.Printf("  - Release: %s\n", releaseName)
			pterm.Printf("  - Chart: %s\n", chartPath)
			pterm.Printf("  - Namespace: %s\n", configs.Namespace)
			pterm.Printf("  - Timeout: %v\n", timeoutDuration)
			pterm.Printf("  - Atomic: %t\n", configs.Atomic)
			pterm.Printf("  - Create Namespace: %t\n", createNamespace)
			pterm.Printf("  - Install if not present: %t\n", installIfNotPresent)
			pterm.Printf("  - Wait: %t\n", wait)
			pterm.Printf("  - Set values: %v\n", configs.Set)
			pterm.Printf("  - Values files: %v\n", configs.File)
			pterm.Printf("  - Set literal: %v\n", configs.SetLiteral)
			pterm.Printf("  - Repo URL: %s\n", RepoURL)
			pterm.Printf("  - Version: %s\n", Version)
			pterm.Printf("  - History Max: %d\n", historyMax)
		}

		// Check if release exists
		exists, err := helm.HelmReleaseExists(releaseName, configs.Namespace, configs.Debug, useAI)
		if err != nil {
			return err
		}

		if !exists {
			if installIfNotPresent {
				if configs.Debug {
					pterm.Println("Release not found, installing...")
				}
				if err := helm.HelmInstall(releaseName, chartPath, configs.Namespace, configs.File, timeoutDuration, configs.Atomic, configs.Debug, configs.Set, configs.SetLiteral, RepoURL, Version, wait, useAI); err != nil {
					return err
				}
				if configs.Debug {
					pterm.Println("Installation completed successfully")
				}
				pterm.Success.Println(".")
				return nil
			} else {
				return fmt.Errorf("release %s not found in namespace %s. Use --install flag to install it", releaseName, configs.Namespace)
			}
		}

		if configs.Debug {
			pterm.Println("Release exists, proceeding with upgrade")
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
			if configs.Debug {
				pterm.Printf("Using default namespace: %s\n", configs.Namespace)
			}
		}

		if configs.Debug {
			pterm.Println("Starting Helm upgrade...")
		}

		err = helm.HelmUpgrade(
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
			wait,
			historyMax,
			useAI,
		)
		if err != nil {
			os.Exit(1)
		}

		return nil
	},
	Example: `
			# Upgrade without waiting (default behavior)
			smurf selm upgrade my-release ./mychart

			# Upgrade and wait for resources to be ready
			smurf selm upgrade my-release ./mychart --wait

			# Upgrade with custom timeout and waiting
			smurf selm upgrade my-release ./mychart --timeout 600 --wait

			# Install if not present without waiting (default)
			smurf selm upgrade my-release ./mychart --install

			# Install if not present with waiting
			smurf selm upgrade my-release ./mychart --install --wait

			# Upgrade with limited history
			smurf selm upgrade my-release ./mychart --history-max 5

			# Upgrade with all options
			smurf selm upgrade my-release ./mychart --wait --timeout 300 --history-max 3 --atomic
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
	upgradeCmd.Flags().BoolVar(&configs.Wait, "wait", false, "Wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment are ready before marking success")
	upgradeCmd.Flags().IntVar(&historyMax, "history-max", 10, "Limit the maximum number of revisions saved per release")
	upgradeCmd.PersistentFlags().BoolVar(&useAI, "ai", false, "Enable AI help mode")
	selmCmd.AddCommand(upgradeCmd)
}
