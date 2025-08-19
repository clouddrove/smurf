package helm

import (
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
)

// HelmReleaseExists checks if a Helm release with the given name exists in the specified namespace.
// It initializes Helm's action configuration, runs the Helm list command, and returns true if a release
// with the given name is found in the specified namespace.
// Helper function to check if release exists
func HelmReleaseExists(releaseName, namespace string, debug bool) (bool, error) {
	if debug {
		pterm.Printf("Checking if release %s exists in namespace %s\n", releaseName, namespace)
	}

	actionConfig, err := initActionConfig(namespace, debug)
	if err != nil {
		return false, err
	}

	listAction := action.NewList(actionConfig)
	listAction.AllNamespaces = false
	listAction.SetStateMask()

	releases, err := listAction.Run()
	if err != nil {
		if debug {
			pterm.Printf("Failed to list releases: %v\n", err)
		}
		return false, err
	}

	for _, r := range releases {
		if r.Name == releaseName && r.Namespace == namespace {
			if debug {
				pterm.Printf("Release found: %s (status: %s)\n", releaseName, r.Info.Status)
			}
			return true, nil
		}
	}

	if debug {
		pterm.Printf("Release not found: %s\n", releaseName)
	}
	return false, nil
}
