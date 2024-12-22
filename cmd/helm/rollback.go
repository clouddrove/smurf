package helm

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

var (
	rollbackOpts = helm.RollbackOptions{
		Namespace: "default",
		Debug:     false,
		Force:     false,
		Timeout:   300,
		Wait:      true,
	}
)

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

			if rollbackOpts.Namespace == "default" && data.Selm.Namespace != "" {
				rollbackOpts.Namespace = data.Selm.Namespace
			}
		}

		if releaseName == "" || revision < 1 {
			return errors.New(color.RedString("RELEASE and REVISION must be provided either as arguments or in the config"))
		}

		if rollbackOpts.Namespace == "" {
			rollbackOpts.Namespace = "default"
		}

		err := helm.HelmRollback(releaseName, revision, rollbackOpts)
		if err != nil {
			return fmt.Errorf(color.RedString("Helm rollback failed: %v", err))
		}
		pterm.Success.Println(fmt.Sprintf("Successfully rolled back release '%s' to revision '%d'", releaseName, revision))
		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringVarP(&rollbackOpts.Namespace, "namespace", "n", rollbackOpts.Namespace, "Namespace of the release")
	rollbackCmd.Flags().BoolVar(&rollbackOpts.Debug, "debug", rollbackOpts.Debug, "Enable debug logging")
	rollbackCmd.Flags().BoolVar(&rollbackOpts.Force, "force", rollbackOpts.Force, "Force rollback even if there are conflicts")
	rollbackCmd.Flags().IntVar(&rollbackOpts.Timeout, "timeout", rollbackOpts.Timeout, "Timeout for the rollback operation in seconds")
	rollbackCmd.Flags().BoolVar(&rollbackOpts.Wait, "wait", rollbackOpts.Wait, "Wait until all resources are rolled back successfully")
	selmCmd.AddCommand(rollbackCmd)
}
