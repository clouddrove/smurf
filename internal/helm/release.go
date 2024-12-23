package helm

import "helm.sh/helm/v3/pkg/action"

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

	return false, nil
}
