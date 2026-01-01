package helm

import (
	"fmt"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
)

// HelmStatus retrieves and displays the status of a specified Helm release within a given namespace.
// It initializes the Helm action configuration, fetches the release status, and presents it in a formatted table.
// Additionally, it checks the readiness of the associated Kubernetes resources and provides detailed feedback.
func HelmStatus(releaseName, namespace string, useAI bool) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Retrieving status for release: %s", releaseName))
	defer spinner.Stop()

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		logDetailedError("helm status", err, namespace, releaseName)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	statusAction := action.NewStatus(actionConfig)
	rel, err := statusAction.Run(releaseName)
	if err != nil {
		logDetailedError("helm status", err, namespace, releaseName)
		ai.AIExplainError(useAI, err.Error())
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
		pterm.Info.Printfln("NOTES: %s\n", rel.Info.Notes)
	} else {
		pterm.Info.Printfln("No additional notes provided for this release.\n")
	}

	printResourcesFromRelease(rel)

	resources, err := parseResourcesFromManifest(rel.Manifest)
	if err != nil {
		pterm.Error.Printfln("Error parsing manifest for readiness check: %v \n", err)
		ai.AIExplainError(useAI, err.Error())
		return nil
	}

	clientset, err := getKubeClient()
	if err != nil {
		pterm.Error.Printfln("Error getting kube client for readiness check: %v \n", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	allReady, notReadyResources, err := resourcesReady(clientset, rel.Namespace, resources)
	if err != nil {
		pterm.Error.Printfln("Error checking resource readiness: %v \n", err)
		ai.AIExplainError(useAI, err.Error())
		return err
	}

	if !allReady {
		pterm.Info.Printfln("Some resources are not ready: \n")
		for _, nr := range notReadyResources {
			pterm.Info.Printfln("- %s \n", nr)
		}
		describeFailedResources(rel.Namespace, rel.Name)
	} else {
		pterm.Success.Printfln("All resources for release '%s' are ready.\n", rel.Name)
	}

	spinner.Success(fmt.Sprintf("Status retrieved successfully for release: %s \n", releaseName))
	return nil
}
