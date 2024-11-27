package helm

import (
	"fmt"
	"strconv"

	"github.com/clouddrove/smurf/configs"
	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

var rollbackAuto bool

var rollbackOpts = helm.RollbackOptions{
	Namespace: "default",
	Debug:     false,
	Force:     false,
	Timeout:   300,
	Wait:      true,
}

// rollbackCmd represents the rollback command of helm
var rollbackCmd = &cobra.Command{
	Use:   "rollback RELEASE REVISION",
	Short: "Roll back a release to a previous revision",
	Long: `Roll back a release to a previous revision. 
The first argument is the name of the release to roll back, and the second is the revision number to roll back to.`,
	Example: ` 
  smurf helm rollback nginx 2
  smurf helm rollback nginx 2 --namespace mynamespace --debug
  smurf helm rollback nginx 2 --force --timeout 600`,
	Args: func(cmd *cobra.Command, args []string) error {

		if len(args) != 2 {
			return fmt.Errorf("exactly two arguments (release name and revision) are required")
		}

		revision, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid revision number '%s': %v", args[1], err)
		}
		if revision < 1 {
			return fmt.Errorf("revision must be a positive integer")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		if rollbackAuto {
			data, err := configs.LoadConfig(configs.FileName)
			if err != nil {
				return err
			}

			args = append(args, data.Selm.ReleaseName, data.Selm.ChartName)

			options := helm.RollbackOptions{
				Namespace: data.Selm.Namespace,
				Debug:     rollbackOpts.Debug,
				Force:     rollbackOpts.Force,
				Timeout:   rollbackOpts.Timeout,
				Wait:      rollbackOpts.Wait,
			}

			if err := helm.HelmRollback(args[0], data.Selm.Revision, options); err != nil {
				return fmt.Errorf("failed to roll back release: %v", err)
			}

			fmt.Printf("Successfully rolled back release '%s' to revision '%d'\n", args[0], 1)
			return nil
		}

		releaseName := args[0]
		revision, _ := strconv.Atoi(args[1])

		options := helm.RollbackOptions{
			Namespace: rollbackOpts.Namespace,
			Debug:     rollbackOpts.Debug,
			Force:     rollbackOpts.Force,
			Timeout:   rollbackOpts.Timeout,
			Wait:      rollbackOpts.Wait,
		}

		if err := helm.HelmRollback(releaseName, revision, options); err != nil {
			return fmt.Errorf("failed to roll back release: %v", err)
		}

		fmt.Printf("Successfully rolled back release '%s' to revision '%d'\n", releaseName, revision)
		return nil
	},
}

func init() {
	rollbackCmd.Flags().StringVarP(&rollbackOpts.Namespace, "namespace", "n", rollbackOpts.Namespace, "Namespace of the release")
	rollbackCmd.Flags().BoolVar(&rollbackOpts.Debug, "debug", rollbackOpts.Debug, "Enable debug logging")
	rollbackCmd.Flags().BoolVar(&rollbackOpts.Force, "force", rollbackOpts.Force, "Force rollback even if there are conflicts")
	rollbackCmd.Flags().IntVar(&rollbackOpts.Timeout, "timeout", rollbackOpts.Timeout, "Timeout for the rollback operation in seconds")
	rollbackCmd.Flags().BoolVar(&rollbackOpts.Wait, "wait", rollbackOpts.Wait, "Wait until all resources are rolled back successfully")
	rollbackCmd.Flags().BoolVarP(&rollbackAuto, "auto", "a", false, "Rollback Helm release automatically")

	selmCmd.AddCommand(rollbackCmd)
}
