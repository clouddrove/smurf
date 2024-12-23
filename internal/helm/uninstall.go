package helm

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
)

func HelmUninstall(releaseName, namespace string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Starting Helm Uninstall for release: %s", releaseName))
	defer spinner.Stop()

	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		logDetailedError("helm uninstall", err, namespace, releaseName)
		return err
	}

	statusAction := action.NewStatus(actionConfig)
	rel, preErr := statusAction.Run(releaseName)

	if preErr == nil && rel != nil {
		printResourcesFromRelease(rel)
	} else {
		color.Yellow("Could not retrieve release '%s' status before uninstall: %v \n", releaseName, preErr)
	}

	client := action.NewUninstall(actionConfig)
	if client == nil {
		err := fmt.Errorf("failed to create Helm uninstall client")
		logDetailedError("helm uninstall", err, namespace, releaseName)
		return err
	}

	resp, err := client.Run(releaseName)
	if err != nil {
		logDetailedError("helm uninstall", err, namespace, releaseName)
		return err
	}

	color.Green("Uninstall Completed Successfully for release: %s \n", releaseName)

	var resources []Resource
	if len(resources) == 0 && resp != nil && resp.Release != nil {
		rs, err := parseResourcesFromManifest(resp.Release.Manifest)
		if err == nil {
			resources = rs
		} else {
			color.Yellow("Could not parse manifest from uninstall response for release '%s': %v \n", releaseName, err)
		}
	}

	if resp != nil && resp.Release != nil {
		color.Cyan("Detailed Information After Uninstall: \n")
		printResourcesFromRelease(resp.Release)
	}

	if len(resources) > 0 {
		color.Cyan("----- RESOURCES REMOVED ----- \n")
		clientset, getErr := getKubeClient()
		if getErr != nil {
			color.Yellow("Could not verify resource removal due to kubeclient error: %v \n", getErr)
			for _, r := range resources {
				color.Green("%s: %s (Assumed Removed) \n", r.Kind, r.Name)
			}
		} else {
			for _, r := range resources {
				removed := resourceRemoved(clientset, namespace, r)
				if removed {
					color.Green("%s: %s (Removed) \n", r.Kind, r.Name)
				} else {
					color.Yellow("%s: %s might still exist. Check your cluster. \n", r.Kind, r.Name)
				}
			}
		}
		color.Cyan("-------------------------------- \n")
	} else {
		color.Green("No resources recorded for this release or unable to parse manifest. Assuming all are removed. \n")
	}

	color.Green("All resources associated with release '%s' have been removed (or no longer found). \n", releaseName)
	return nil
}
