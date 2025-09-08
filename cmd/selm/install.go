package selm

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var RepoURL string
var Version string

var installCmd = &cobra.Command{
	Use:   "install [RELEASE] [CHART]",
	Short: "Install a Helm chart into a Kubernetes cluster.",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var releaseName, chartPath string
		if len(args) >= 1 {
			releaseName = args[0]
		}
		if len(args) >= 2 {
			chartPath = args[1]
		}

		// Step 1: Load configuration
		if releaseName == "" || chartPath == "" {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				pterm.Error.WithShowLineNumber(true).Printfln("❌ Failed to load config: %v", err)
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
				pterm.Error.Println("❌ Both RELEASE and CHART must be provided either as arguments or in the config")
				return fmt.Errorf("both RELEASE and CHART must be provided either as arguments or in the config")
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		timeoutDuration := time.Duration(configs.Timeout) * time.Second

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		// Display configuration summary
		if configs.Debug {
			pterm.EnableDebugMessages()
			pterm.Debug.Printfln("🔍 Starting Helm install with configuration:")
			pterm.Debug.Printfln("  🏷️  Release: %s", releaseName)
			pterm.Debug.Printfln("  📦 Chart: %s", chartPath)
			pterm.Debug.Printfln("  📁 Namespace: %s", configs.Namespace)
			pterm.Debug.Printfln("  ⏱️  Timeout: %v", timeoutDuration)
			pterm.Debug.Printfln("  📄 Values files: %v", configs.File)
			pterm.Debug.Printfln("  ⚙️  Set values: %v", configs.Set)
			pterm.Debug.Printfln("  🔧 Set literal values: %v", configs.SetLiteral)
			pterm.Debug.Printfln("  🌐 Repo URL: %s", RepoURL)
			pterm.Debug.Printfln("  🏷️  Version: %s", Version)
		} else {
			// Create configuration summary table
			configData := pterm.TableData{
				{"Setting", "Value"},
				{"🏷️  Release", releaseName},
				{"📦 Chart", chartPath},
				{"📁 Namespace", configs.Namespace},
				{"⏱️  Timeout", timeoutDuration.String()},
				{"📄 Values Files", fmt.Sprintf("%v", configs.File)},
			}

			if RepoURL != "" {
				configData = append(configData, []string{"🌐 Repo URL", RepoURL})
			}
			if Version != "" {
				configData = append(configData, []string{"🏷️  Version", Version})
			}
			if configs.Atomic {
				configData = append(configData, []string{"⚡ Atomic", "true"})
			}
			if len(configs.Set) > 0 {
				configData = append(configData, []string{"⚙️  Set Values", fmt.Sprintf("%v", configs.Set)})
			}
			if len(configs.SetLiteral) > 0 {
				configData = append(configData, []string{"🔧 Set Literal", fmt.Sprintf("%v", configs.SetLiteral)})
			}

			pterm.DefaultSection.Println("📋 Configuration Summary")
			configSummary := pterm.DefaultTable.WithBoxed(true).WithData(configData)
			configSummary.Render()
			fmt.Println()
		}

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
			pterm.Error.WithShowLineNumber(true).Printfln("💥 Installation failed: %v", err)
			return fmt.Errorf("installation failed: %w", err)
		}
		return nil
	},
	Example: `...`, // Keep your existing example
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
