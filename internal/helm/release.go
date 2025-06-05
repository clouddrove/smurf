package helm

import (
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
)

// HelmReleaseExists checks if a Helm release with the given name exists in the specified namespace.
// It initializes Helm's action configuration, runs the Helm list command, and returns true if a release
// with the given name is found in the specified namespace.
func HelmReleaseExists(releaseName, namespace string) (bool, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		return false, err
	}

	list := action.NewList(actionConfig)
	list.Deployed = true
	list.AllNamespaces = false
	releases, err := list.Run()
	if err != nil {
		return false, err
	}

	for _, rel := range releases {
		if rel.Name == releaseName && rel.Namespace == namespace {
			return true, nil
		}
	}

	pterm.Success.Printfln("Helm realese exists...")
	return false, nil
}
