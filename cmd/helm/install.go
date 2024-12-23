package helm

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

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
				return errors.New(color.RedString("both RELEASE and CHART must be provided either as arguments or in the config"))
			}

			if configs.Namespace == "" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if releaseName == "" || chartPath == "" {
			return errors.New(color.RedString("RELEASE and CHART must be provided"))
		}

		timeoutDuration := time.Duration(configs.Timeout) * time.Second

		buildArgsMap := make(map[string]string)
		for _, arg := range configs.Set {
			parts := splitKeyValue(arg)
			if len(parts) == 2 {
				buildArgsMap[parts[0]] = parts[1]
			}
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		pterm.Info.Println("Starting Helm install...")
		err := helm.HelmInstall(releaseName, chartPath, configs.Namespace, configs.File, timeoutDuration, configs.Atomic, configs.Debug, configs.Set)
		if err != nil {
			return errors.New(color.RedString("Helm install failed: %v", err))
		}
		pterm.Success.Println("Helm chart installed successfully.")
		return nil
	},
	Example: `
  smurf selm install my-release ./mychart
  smurf selm install my-release ./mychart -n my-namespace
  smurf selm install my-release ./mychart -f values.yaml
  smurf selm install my-release ./mychart --timeout=600
  smurf selm install
  # In the last example, it will read RELEASE and CHART from the config file
  `,
}

func splitKeyValue(arg string) []string {
	parts := make([]string, 2)
	for i, part := range []rune(arg) {
		if part == '=' {
			parts[0] = string([]rune(arg)[:i])
			parts[1] = string([]rune(arg)[i+1:])
			break
		}
	}
	return parts
}

func init() {
	installCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "", "Specify the namespace to install the Helm chart")
	installCmd.Flags().IntVar(&configs.Timeout, "timeout", 150, "Specify the timeout in seconds to wait for any individual Kubernetes operation")
	installCmd.Flags().StringArrayVarP(&configs.File, "values", "f", []string{}, "Specify values in a YAML file")
	installCmd.Flags().BoolVar(&configs.Atomic, "atomic", false, "If set, installation process purges chart on fail")
	installCmd.Flags().BoolVar(&configs.Debug, "debug", false, "Enable verbose output")
	installCmd.Flags().StringSliceVar(&configs.Set, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	selmCmd.AddCommand(installCmd)
}
