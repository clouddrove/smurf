package helm

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func HelmRollback(releaseName string, revision int, opts RollbackOptions) error {
	if releaseName == "" {
		return fmt.Errorf("release name cannot be empty \n")
	}
	if revision < 1 {
		return fmt.Errorf("revision must be a positive integer \n")
	}

	color.Green("Starting Helm Rollback for release: %s to revision %d \n", releaseName, revision)

	settings := cli.New()
	settings.Debug = opts.Debug

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), opts.Namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		if settings.Debug {
			fmt.Printf(format, v...)
		}
	}); err != nil {
		logDetailedError("helm rollback", err, opts.Namespace, releaseName)
		return fmt.Errorf("failed to initialize Helm action configuration: %w \n", err)
	}

	rollbackAction := action.NewRollback(actionConfig)
	rollbackAction.Version = revision
	rollbackAction.Force = opts.Force
	rollbackAction.Timeout = time.Duration(opts.Timeout) * time.Second
	rollbackAction.Wait = opts.Wait

	if err := rollbackAction.Run(releaseName); err != nil {
		logDetailedError("helm rollback", err, opts.Namespace, releaseName)
		return err
	}

	if err := HelmStatus(releaseName, opts.Namespace); err != nil {
		color.Yellow("Rollback completed, but status retrieval failed. Check the release status manually.\n")
		return nil
	}

	color.Green("Rollback Completed Successfully for release: %s to revision %d \n", releaseName, revision)
	return nil
}
