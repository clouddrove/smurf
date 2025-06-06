package helm

import (
	"fmt"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
)

// HelmList retrieves and displays all Helm releases in the specified namespace (with AllNamespaces = true).
// It initializes Helm's action configuration, runs the Helm list command, and prints release details in a
// formatted table. If an error occurs, it logs the failure; otherwise, it returns the list of releases.
func HelmList(namespace string) ([]*release.Release, error) {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Listing releases in namespace: %s", namespace))
	defer spinner.Stop()

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		pterm.Error.Printfln("Failed to initialize action configuration: %v \n", err)
		return nil, err
	}

	client := action.NewList(actionConfig)
	client.AllNamespaces = true

	releases, err := client.Run()
	if err != nil {
		pterm.Error.Printfln("Failed to list releases: %v \n", err)
		return nil, err
	}

	fmt.Println()
	pterm.FgCyan.Printfln("%-17s %-10s %-8s %-20s %-7s %-30s \n", "NAME", "NAMESPACE", "REVISION", "UPDATED", "STATUS", "CHART")
	for _, rel := range releases {
		updatedStr := rel.Info.LastDeployed.Local().Format("2006-01-02 15:04:05")
		pterm.FgYellow.Printfln("%-17s %-10s %-8d %-20s %-7s %-30s \n",
			rel.Name, rel.Namespace, rel.Version, updatedStr, rel.Info.Status.String(), rel.Chart.Metadata.Name+"-"+rel.Chart.Metadata.Version)
	}

	spinner.Success("Releases listed successfully. \n")
	return releases, nil
}
