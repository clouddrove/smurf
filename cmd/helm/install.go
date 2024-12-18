package helm

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var (
	installNamespace string
	installAuto      bool
	installFiles     []string
	installTimeout   int
	installAtomic    bool
	installdebug     bool
	installSet       []string
)

var installCmd = &cobra.Command{
	Use:   "install [RELEASE] [CHART]",
	Short: "Install a Helm chart into a Kubernetes cluster.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if installAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			if installNamespace == "" {
				installNamespace = data.Selm.Namespace
			}

			releaseName := data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			if len(args) < 2 {
				args = []string{releaseName, data.Selm.ChartName}
			}

			timeoutDuration := time.Duration(installTimeout) * time.Second
			return helm.HelmInstall(args[0], args[1], installNamespace, installFiles, timeoutDuration, installAtomic, installdebug, installSet)
		}

		if len(args) < 2 {
			return errors.New("requires exactly two arguments: [NAME] [DIRECTORY]")
		}

		releaseName := args[0]
		chartPath := args[1]
		if installNamespace == "" {
			installNamespace = "default"
		}

		timeoutDuration := time.Duration(installTimeout) * time.Second
		return helm.HelmInstall(releaseName, chartPath, installNamespace, installFiles, timeoutDuration, installAtomic, installdebug, installSet)
	},
	Example: `
  smurf selm install my-release ./mychart
  smurf selm install my-release ./mychart -n my-namespace
  smurf selm install my-release ./mychart -f values.yaml
  smurf selm install my-release ./mychart --timeout=600
  `,
}

func init() {
	installCmd.Flags().StringVarP(&installNamespace, "namespace", "n", "", "Specify the namespace to install the Helm chart")
	installCmd.Flags().BoolVarP(&installAuto, "auto", "a", false, "Install Helm chart automatically")
	installCmd.Flags().IntVar(&installTimeout, "timeout", 150, "Specify the timeout in seconds to wait for any individual Kubernetes operation")
	installCmd.Flags().StringArrayVarP(&installFiles, "values", "f", []string{}, "Specify values in a YAML file")
	installCmd.Flags().BoolVar(&installAtomic, "atomic", false, "If set, installation process purges chart on fail")
	installCmd.Flags().BoolVar(&installdebug, "debug", false, "Enable verbose output")
	installCmd.Flags().StringSliceVar(&installSet, "set", []string{}, "Set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)")
	selmCmd.AddCommand(installCmd)
}
