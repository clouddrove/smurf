// helm/history.go
package helm

import (
	"fmt"
	"os"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
)

// HelmHistory shows the revision history of a Helm release using pterm.Table.
//
// format selects the output shape: "table" (default) renders the existing
// human-facing table; "json"/"yaml" print the revision list as a single
// machine-readable document to stdout and suppress every other stdout write,
// so pipelines consuming stdout only ever see that document.
func HelmHistory(releaseName, namespace string, max int, format string, useAI bool) error {
	isTable := format == "" || format == "table"

	if !isTable {
		// Helm and shared helpers can print via pterm unconditionally;
		// redirect pterm's default writer to stderr for the duration of the
		// call so nothing lands inside the JSON/YAML document on stdout,
		// and restore it on return.
		pterm.SetDefaultOutput(os.Stderr)
		defer pterm.SetDefaultOutput(os.Stdout)
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		if isTable {
			ai.AIExplainError(useAI, err.Error())
		}
		return fmt.Errorf("failed to initialize Helm action configuration: %v", err)
	}

	client := action.NewHistory(actionConfig)
	client.Max = max

	releases, err := client.Run(releaseName)
	if err != nil {
		if isTable {
			ai.AIExplainError(useAI, err.Error())
		}
		return fmt.Errorf("failed to get release history: %v", err)
	}

	releases = sortReleasesByRevision(releases)

	if !isTable {
		elements := historyElements(releases)
		if format == "yaml" {
			return printYAML(elements)
		}
		return printJSON(elements)
	}

	if len(releases) == 0 {
		pterm.Info.Printfln("No revision history found for release %s", releaseName)
		return nil
	}

	printHistoryTable(releases)
	return nil
}

// historyElements converts release revisions into plain maps for JSON/YAML
// output, always returning a non-nil (possibly empty) slice so the JSON
// document is "[]" rather than "null" when there is no history.
func historyElements(releases []*release.Release) []map[string]interface{} {
	elements := make([]map[string]interface{}, 0, len(releases))
	for _, r := range releases {
		if r == nil {
			continue
		}
		elements = append(elements, map[string]interface{}{
			"revision":    safeInt(r.Version),
			"updated":     safeTime(r.Info),
			"status":      safeStatus(r.Info),
			"chart":       safeChartName(r.Chart),
			"app_version": safeAppVersion(r.Chart),
			"description": safeDescription(r.Info),
		})
	}
	return elements
}

func sortReleasesByRevision(releases []*release.Release) []*release.Release {
	for i := 0; i < len(releases)-1; i++ {
		for j := 0; j < len(releases)-i-1; j++ {
			if releases[j] == nil || releases[j+1] == nil {
				continue
			}
			if releases[j].Version > releases[j+1].Version {
				releases[j], releases[j+1] = releases[j+1], releases[j]
			}
		}
	}
	return releases
}

func printHistoryTable(releases []*release.Release) {
	// Create table data
	tableData := [][]string{
		{"REVISION", "UPDATED", "STATUS", "CHART", "APP VERSION", "DESCRIPTION"},
	}

	for _, r := range releases {
		if r == nil {
			continue
		}

		tableData = append(tableData, []string{
			fmt.Sprintf("%d", safeInt(r.Version)),
			safeTime(r.Info),
			string(r.Info.Status),
			safeChartName(r.Chart),
			safeAppVersion(r.Chart),
			truncateDescription(safeDescription(r.Info), 30),
		})
	}

	// Create and render table
	table := pterm.DefaultTable.
		WithHasHeader(true).
		WithBoxed(true).
		WithData(tableData)

	err := table.Render()
	if err != nil {
		pterm.Error.Printfln("Failed to render table: %v", err)
		// Fallback to simple output
		for _, row := range tableData {
			fmt.Println(row)
		}
	}
}

// Helper functions with pterm styling
func colorizeStatus(status string) string {
	switch status {
	case "deployed":
		return pterm.LightGreen(status)
	case "failed":
		return pterm.LightRed(status)
	case "pending":
		return pterm.LightYellow(status)
	default:
		return pterm.LightCyan(status)
	}
}

func truncateDescription(desc string, maxLen int) string {
	if len(desc) <= maxLen {
		return desc
	}
	return desc[:maxLen-3] + "..."
}

// Existing safe accessor functions remain the same
func safeInt(version int) int {
	if version < 0 {
		return 0
	}
	return version
}

func safeTime(info *release.Info) string {
	if info == nil || info.LastDeployed.IsZero() {
		return "unknown"
	}
	return info.LastDeployed.Format("2006-01-02 15:04:05")
}

func safeStatus(info *release.Info) string {
	if info == nil || info.Status == "" {
		return "unknown"
	}
	return string(info.Status)
}

func safeChartName(chart *chart.Chart) string {
	if chart == nil || chart.Metadata == nil {
		return "unknown"
	}
	return fmt.Sprintf("%s-%s", chart.Metadata.Name, chart.Metadata.Version)
}

func safeAppVersion(chart *chart.Chart) string {
	if chart == nil || chart.Metadata == nil {
		return "unknown"
	}
	return chart.Metadata.AppVersion
}

func safeDescription(info *release.Info) string {
	if info == nil {
		return "unknown"
	}
	return info.Description
}
