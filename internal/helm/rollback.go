package helm

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// HelmRollback performs a rollback of a specified Helm release to a given revision.
// It initializes the Helm action configuration, sets up the rollback parameters,
// executes the rollback, and then retrieves the status of the release post-rollback.
// Detailed error logging is performed if any step fails.
func HelmRollback(releaseName string, revision int, opts RollbackOptions, historyMax int, useAI bool) error {
	if releaseName == "" {
		pterm.Error.Printfln("release name cannot be empty")
		return errors.New("release name cannot be empty")
	}
	if revision < 1 {
		pterm.Error.Printfln("revision must be a positive integer")
		return errors.New("revision must be a positive integer")
	}

	pterm.Success.Printfln("Starting Helm Rollback for release: %s to revision %d \n", releaseName, revision)

	settings := cli.New()
	settings.Debug = opts.Debug

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), opts.Namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		if settings.Debug {
			fmt.Printf(format, v...)
		}
	}); err != nil {
		logDetailedError("helm rollback", err, opts.Namespace, releaseName)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to initialize Helm action configuration: %v", err)
	}

	rollbackAction := action.NewRollback(actionConfig)
	rollbackAction.Version = revision
	rollbackAction.Force = opts.Force
	rollbackAction.Timeout = time.Duration(opts.Timeout) * time.Second
	rollbackAction.Wait = opts.Wait
	rollbackAction.MaxHistory = historyMax

	if err := rollbackAction.Run(releaseName); err != nil {
		logDetailedError("helm rollback", err, opts.Namespace, releaseName)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	if err := HelmStatus(releaseName, opts.Namespace, useAI); err != nil {
		pterm.FgYellow.Print("Rollback completed, but status retrieval failed. Check the release status manually.\n")
		ai.AIExplainError(useAI, err.Error())
		return nil
	}

	pterm.FgGreen.Printfln("Rollback Completed Successfully for release: %s to revision %d \n", releaseName, revision)
	return nil
}
