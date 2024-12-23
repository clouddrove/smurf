package helm

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
)

func HelmStatus(releaseName, namespace string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Retrieving status for release: %s", releaseName))
	defer spinner.Stop()

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		logDetailedError("helm status", err, namespace, releaseName)
		return err
	}

	statusAction := action.NewStatus(actionConfig)
	rel, err := statusAction.Run(releaseName)
	if err != nil {
		logDetailedError("helm status", err, namespace, releaseName)
		return err
	}

	data := [][]string{
		{"NAME", rel.Name},
		{"NAMESPACE", rel.Namespace},
		{"STATUS", rel.Info.Status.String()},
		{"REVISION", fmt.Sprintf("%d", rel.Version)},
		{"TEST SUITE", "None"},
	}

	pterm.DefaultTable.WithHasHeader(false).WithData(data).Render()

	if rel.Info.Notes != "" {
		color.Green("NOTES: %s\n", rel.Info.Notes)
	} else {
		color.Yellow("No additional notes provided for this release.\n")
	}

	printResourcesFromRelease(rel)

	resources, err := parseResourcesFromManifest(rel.Manifest)
	if err != nil {
		color.Red("Error parsing manifest for readiness check: %v \n", err)
		return nil
	}

	clientset, err := getKubeClient()
	if err != nil {
		color.Red("Error getting kube client for readiness check: %v \n", err)
		return err
	}

	allReady, notReadyResources, err := resourcesReady(clientset, rel.Namespace, resources)
	if err != nil {
		color.Red("Error checking resource readiness: %v \n", err)
		return err
	}

	if !allReady {
		color.Yellow("Some resources are not ready: \n")
		for _, nr := range notReadyResources {
			color.Yellow("- %s \n", nr)
		}
		describeFailedResources(rel.Namespace, rel.Name)
	} else {
		color.Green("All resources for release '%s' are ready.\n", rel.Name)
	}

	spinner.Success(fmt.Sprintf("Status retrieved successfully for release: %s \n", releaseName))
	return nil
}
