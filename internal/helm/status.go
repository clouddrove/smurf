package helm

import (
	"fmt"
	"os"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
)

// HelmStatus retrieves and displays the status of a specified Helm release within a given namespace.
// It initializes the Helm action configuration, fetches the release status, and presents it in a formatted table.
// Additionally, it checks the readiness of the associated Kubernetes resources and provides detailed feedback.
//
// format selects the output shape: "table" (default) renders the existing
// human-facing table, spinner, notes and readiness report; "json"/"yaml"
// print a single machine-readable document to stdout and suppress every
// other stdout write (spinner, tables, notes, AI explanations) so pipelines
// consuming stdout only ever see that document. Errors are still returned
// normally and land on stderr via the caller.
func HelmStatus(releaseName, namespace, format string, useAI bool) error {
	isTable := format == "" || format == "table"

	if !isTable {
		// Shared helpers reached below (getKubeClient, resourcesReady) print
		// via pterm unconditionally; redirect pterm's default writer to
		// stderr for the duration of the call so none of that can land
		// inside the JSON/YAML document on stdout, and restore it on return.
		pterm.SetDefaultOutput(os.Stderr)
		defer pterm.SetDefaultOutput(os.Stdout)
	}

	var spinner *pterm.SpinnerPrinter
	if isTable {
		spinner, _ = pterm.DefaultSpinner.Start(fmt.Sprintf("Retrieving status for release: %s", releaseName))
		defer spinner.Stop()
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		if isTable {
			logDetailedError("helm status", err, namespace, releaseName)
			ai.AIExplainError(useAI, err.Error())
		}
		return err
	}

	statusAction := action.NewStatus(actionConfig)
	rel, err := statusAction.Run(releaseName)
	if err != nil {
		if isTable {
			logDetailedError("helm status", err, namespace, releaseName)
			ai.AIExplainError(useAI, err.Error())
		}
		return err
	}

	if isTable {
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
	}

	resources, err := parseResourcesFromManifest(rel.Manifest)
	if err != nil {
		if isTable {
			pterm.Error.Printfln("Error parsing manifest for readiness check: %v \n", err)
			ai.AIExplainError(useAI, err.Error())
			return nil
		}
		return fmt.Errorf("error parsing manifest for readiness check: %w", err)
	}

	clientset, err := getKubeClient()
	if err != nil {
		if isTable {
			pterm.Error.Printfln("Error getting kube client for readiness check: %v \n", err)
			ai.AIExplainError(useAI, err.Error())
		}
		return err
	}

	allReady, notReadyResources, err := resourcesReady(clientset, rel.Namespace, resources)
	if err != nil {
		if isTable {
			pterm.Error.Printfln("Error checking resource readiness: %v \n", err)
			ai.AIExplainError(useAI, err.Error())
		}
		return err
	}

	if !isTable {
		if notReadyResources == nil {
			notReadyResources = []string{}
		}
		result := map[string]interface{}{
			"name":                rel.Name,
			"namespace":           rel.Namespace,
			"status":              rel.Info.Status.String(),
			"revision":            rel.Version,
			"notes":               rel.Info.Notes,
			"all_resources_ready": allReady,
			"not_ready_resources": notReadyResources,
		}
		if format == "yaml" {
			return printYAML(result)
		}
		return printJSON(result)
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
