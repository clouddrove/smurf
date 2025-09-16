package selm

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var RepoURL string
var Version string

var installCmd = &cobra.Command{
	Use:          "install [RELEASE] [CHART]",
	Short:        "Install a Helm chart into a Kubernetes cluster.",
	Args:         cobra.MaximumNArgs(2),
	SilenceUsage: true,
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
				return fmt.Errorf("❌ Failed to load config: %w", err)
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
				return fmt.Errorf("❌ Both RELEASE and CHART must be provided either as arguments or in the config")
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		timeoutDuration := time.Duration(configs.Timeout) * time.Second

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		if configs.Debug {
			fmt.Printf("🔍 Debug: Starting Helm install with configuration:\n")
			fmt.Printf("  Release: %s\n", releaseName)
			fmt.Printf("  Chart: %s\n", chartPath)
			fmt.Printf("  Namespace: %s\n", configs.Namespace)
			fmt.Printf("  Timeout: %v\n", timeoutDuration)
			fmt.Printf("  Values files: %v\n", configs.File)
			fmt.Printf("  Set values: %v\n", configs.Set)
			fmt.Printf("  Set literal values: %v\n", configs.SetLiteral)
			fmt.Printf("  Repo URL: %s\n", RepoURL)
			fmt.Printf("  Version: %s\n", Version)
		}

		fmt.Printf("🚀 Installing release '%s' in namespace '%s'\n", releaseName, configs.Namespace)

		// Create progress tracker
		err := helm.HelmInstall(
			releaseName,
			chartPath,
			configs.Namespace,
			configs.File,
			timeoutDuration,
			configs.Atomic,
			configs.Debug,
			configs.Set,
			configs.SetLiteral,
			RepoURL,
			Version,
		)
		if err != nil {
			// Use the structured error formatting
			fmt.Print(helm.FormatError(err))

			// Provide troubleshooting tips based on error type
			if strings.Contains(err.Error(), "TLS handshake timeout") {
				fmt.Println(pterm.Cyan("💡 Troubleshooting Tips:"))
				fmt.Println("├── Check if your Kubernetes cluster is running")
				fmt.Println("├── Verify your kubeconfig context is correct")
				fmt.Println("└── Ensure network connectivity to the cluster")
			} else if strings.Contains(err.Error(), "not found") {
				fmt.Println(pterm.Cyan("💡 Troubleshooting Tips:"))
				fmt.Println("├── Verify the chart name or repository URL")
				fmt.Println("├── Check if the chart version exists")
				fmt.Println("└── Ensure you have access to the repository")
			}

			return fmt.Errorf("installation failed")
		}

		return nil
	},
	Example: `
  smurf selm install my-release ./mychart
  smurf selm install my-release ./mychart -n my-namespace
  smurf selm install my-release ./mychart -f values.yaml
  smurf selm install my-release ./mychart --timeout=600
  smurf selm install prometheus-11 prometheus --repo https://prometheus-community.github.io/helm-charts --version 13.0.0
  smurf selm install prometheus prometheus-community/prometheus
  smurf selm install my-release ./mychart --set key1=val1 --set key2=val2
  smurf selm install my-release ./mychart --set-literal myPassword='MySecurePass!'
  smurf selm install
  # In the last example, it will read RELEASE and CHART from the config file
  `,
}

func init() {
	installCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "Specify the namespace to install the Helm chart")
	installCmd.Flags().IntVar(&configs.Timeout, "timeout", 600, "Specify the timeout in seconds to wait for any individual Kubernetes operation")
	installCmd.Flags().StringArrayVarP(&configs.File, "values", "f", []string{}, "Specify values in a YAML file")
	installCmd.Flags().BoolVar(&configs.Atomic, "atomic", false, "If set, installation process purges chart on fail")
	installCmd.Flags().BoolVar(&configs.Debug, "debug", false, "Enable verbose output")
	installCmd.Flags().StringSliceVar(&configs.Set, "set", []string{}, "Set values on the command line")
	installCmd.Flags().StringSliceVar(&configs.SetLiteral, "set-literal", []string{}, "Set literal values on the command line")
	installCmd.Flags().StringVar(&RepoURL, "repo", "", "Specify the chart repository URL for remote charts")
	installCmd.Flags().StringVar(&Version, "version", "", "Specify the chart version to install")
	selmCmd.AddCommand(installCmd)
}
