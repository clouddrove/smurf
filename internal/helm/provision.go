package helm

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"helm.sh/helm/v3/pkg/action"
)

// HelmProvision orchestrates a "provisioning" workflow that checks whether a specified Helm release
// already exists in the cluster and chooses either to install or upgrade. In parallel, it also runs
// linting and template rendering for the chart. If any step fails, a consolidated error is returned.
// Otherwise, a success message is printed.
func HelmProvision(releaseName, chartPath, namespace string) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		color.Red("Failed to initialize Helm action configuration: %v \n", err)
		return err
	}

	client := action.NewList(actionConfig)
	results, err := client.Run()
	if err != nil {
		color.Red("Failed to list releases: %v \n", err)
		return err
	}

	var wg sync.WaitGroup
	var installErr, upgradeErr, lintErr, templateErr error
	exists := false

	for _, result := range results {
		if result.Name == releaseName {
			exists = true
			break
		}
	}

	wg.Add(1)
	if exists {
		go func() {
			defer wg.Done()
			// Updated HelmUpgrade call with the new parameter signature
			upgradeErr = HelmUpgrade(
				releaseName, 
				chartPath, 
				namespace, 
				nil,         // setValues
				nil,         // valuesFiles
				nil,         // setLiteral (new required parameter)
				false,       // createNamespace
				false,       // atomic
				0,           // timeout as time.Duration
				false,       // debug
				"",          // repoURL
				"",          // version
			)
		}()
	} else {
		go func() {
			defer wg.Done()
			// Note: you're using different parameter types here:
			// timeout as int (300) vs. time.Duration (0) in the upgraded version
			installErr = HelmInstall(releaseName, chartPath, namespace, []string{}, 300*time.Second, false, false, []string{}, []string{}, "", "")
		}()
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		lintErr = HelmLint(chartPath, nil)
	}()
	go func() {
		defer wg.Done()
		templateErr = HelmTemplate(releaseName, chartPath, namespace, "", []string{})
	}()

	wg.Wait()

	if installErr != nil || upgradeErr != nil || lintErr != nil || templateErr != nil {
		if installErr != nil {
			color.Red("Install failed: %v \n", installErr)
		}
		if upgradeErr != nil {
			color.Red("Upgrade failed: %v \n", upgradeErr)
		}
		if lintErr != nil {
			color.Red("Lint failed: %v \n", lintErr)
		}
		if templateErr != nil {
			color.Red("Template rendering failed: %v \n", templateErr)
		}
		return fmt.Errorf("provisioning failed \n")
	}

	color.Green("Provisioning completed successfully. \n")
	return nil
}