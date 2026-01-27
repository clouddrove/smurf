package helm

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
)

// HelmProvision orchestrates a "provisioning" workflow that checks whether a specified Helm release
// already exists in the cluster and chooses either to install or upgrade. In parallel, it also runs
// linting and template rendering for the chart. If any step fails, a consolidated error is returned.
// Otherwise, a success message is printed.
func HelmProvision(releaseName, chartPath, namespace string) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		pterm.Error.Printfln("Failed to initialize Helm action configuration: %v \n", err)
		return err
	}

	// Check if release exists
	exists, err := checkReleaseExists(actionConfig, releaseName)
	if err != nil {
		return err
	}

	// Run lint and template in parallel
	var wg sync.WaitGroup
	var lintErr, templateErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		lintErr = HelmLint(chartPath, nil, false)
	}()
	go func() {
		defer wg.Done()
		templateErr = HelmTemplate(releaseName, chartPath, namespace, "", []string{}, false)
	}()

	// Perform install/upgrade after lint/template
	var operationErr error
	if exists {
		pterm.Info.Printfln("Release %s exists, performing upgrade...", releaseName)

		// First do a dry-run
		dryRunErr := HelmUpgrade(
			releaseName,
			chartPath,
			namespace,
			nil,           // setValues
			nil,           // valuesFiles
			nil,           // setLiteral
			false,         // createNamespace
			true,          // atomic
			5*time.Minute, // timeout
			true,          // dry-run
			"",            // repoURL
			"",            // version
			true,
			5,
			false,
			false,
		)

		if dryRunErr != nil {
			return fmt.Errorf("upgrade dry-run failed: %v", dryRunErr)
		}

		// Actual upgrade
		operationErr = HelmUpgrade(
			releaseName,
			chartPath,
			namespace,
			nil,           // setValues
			nil,           // valuesFiles
			nil,           // setLiteral
			false,         // createNamespace
			true,          // atomic
			5*time.Minute, // timeout
			false,         // dry-run
			"",            // repoURL
			"",            // version
			true,
			5,
			false,
			false,
		)
	} else {
		pterm.Info.Printfln("Release %s does not exist, performing install...", releaseName)
		operationErr = HelmInstall(
			releaseName,
			chartPath,
			namespace,
			[]string{},
			5*time.Minute,
			true, // createNamespace
			true, // atomic
			[]string{},
			[]string{},
			"",
			"",
			true,
			false,
		)
	}
	wg.Wait()

	// Check all errors
	if operationErr != nil {
		pterm.Error.Printfln("Operation failed: %v \n", operationErr)
		return operationErr
	}
	if lintErr != nil {
		pterm.Warning.Printfln("Lint warnings: %v \n", lintErr)
	}
	if templateErr != nil {
		pterm.Warning.Printfln("Template warnings: %v \n", templateErr)
	}

	pterm.Success.Printfln("Provisioning completed successfully for %s in namespace %s", releaseName, namespace)
	return nil
}

func checkReleaseExists(actionConfig *action.Configuration, releaseName string) (bool, error) {
	client := action.NewList(actionConfig)
	client.All = true // Check all namespaces
	results, err := client.Run()
	if err != nil {
		return false, fmt.Errorf("failed to list releases: %v", err)
	}

	for _, result := range results {
		if result.Name == releaseName {
			return true, nil
		}
	}
	return false, nil
}
