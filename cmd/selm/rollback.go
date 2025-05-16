package selm

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// rollbackCmd facilitates rolling back a Helm release to a specified previous revision.
// It either takes both RELEASE and REVISION as command-line arguments or reads them
// from a config file if none are provided. The user can also configure namespace, timeout,
// debug, and force options. If the release or revision is invalid, an error is returned.
var rollbackCmd = &cobra.Command{
	Use:   "rollback [RELEASE] [REVISION]",
	Short: "Roll back a release to a previous revision",
	Long: `Roll back a release to a previous revision.
The first argument is the name of the release to roll back, and the second is the revision number to roll back to.`,
	Example: ` 
      smurf helm rollback nginx 2
      smurf helm rollback nginx 2 --namespace mynamespace --debug
      smurf helm rollback nginx 2 --force --timeout 600
      smurf helm rollback
      # In this example, it will read RELEASE and REVISION from the config file
    `,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 && len(args) != 2 {
			return fmt.Errorf("requires either exactly two arguments (RELEASE and REVISION) or none")
		}

		if len(args) == 2 {
			revision, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid revision number '%s': %v", args[1], err)
			}
			if revision < 1 {
				return fmt.Errorf("revision must be a positive integer")
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var releaseName string
		var revision int

		if len(args) == 2 {
			releaseName = args[0]
			revision, _ = strconv.Atoi(args[1])
		} else {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			releaseName = data.Selm.ReleaseName
			if releaseName == "" {
				releaseName = filepath.Base(data.Selm.ChartName)
			}

			revision = data.Selm.Revision
			if revision < 1 {
				return fmt.Errorf("revision must be a positive integer")
			}

			if configs.Namespace == "default" && data.Selm.Namespace != "" {
				configs.Namespace = data.Selm.Namespace
			}
		}

		if releaseName == "" || revision < 1 {
			return errors.New(color.RedString("RELEASE and REVISION must be provided either as arguments or in the config"))
		}

		if configs.Namespace == "" {
			configs.Namespace = "default"
		}

		rollbackOpts := helm.RollbackOptions{
			Namespace: configs.Namespace,
			Debug:     configs.Debug,
			Force:     configs.Force,
			Timeout:   configs.Timeout,
			Wait:      configs.Wait,
		}

		err := helm.HelmRollback(releaseName, revision, rollbackOpts)
		if err != nil {
			return fmt.Errorf("%v", color.RedString("Helm rollback failed: %v", err))
		}
		pterm.Success.Println(fmt.Sprintf("Successfully rolled back release '%s' to revision '%d'", releaseName, revision))
		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringVarP(&configs.Namespace, "namespace", "n", "default", "Namespace of the release")
	rollbackCmd.Flags().BoolVar(&configs.Debug, "debug", false, "Enable debug logging")
	rollbackCmd.Flags().BoolVar(&configs.Force, "force", false, "Force rollback even if there are conflicts")
	rollbackCmd.Flags().IntVar(&configs.Timeout, "timeout", 300, "Timeout for the rollback operation in seconds")
	rollbackCmd.Flags().BoolVar(&configs.Wait, "wait", true, "Wait until all resources are rolled back successfully")
	selmCmd.AddCommand(rollbackCmd)
}
