package helm

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
)

// ListReleases returns and displays Helm releases with flexible output formats
func ListReleases(namespace, format string) ([]*release.Release, error) {
	cfg := new(action.Configuration)
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		return nil, fmt.Errorf("helm init failed: %w", err)
	}

	client := action.NewList(cfg)
	client.AllNamespaces = namespace == ""
	client.StateMask = action.ListAll // Show all statuses

	releases, err := client.Run()
	if err != nil {
		return nil, fmt.Errorf("release listing failed: %w", err)
	}

	if err := printOutput(releases, format, namespace); err != nil {
		return releases, err // Return releases even if printing fails
	}

	return releases, nil
}

func printOutput(releases []*release.Release, format, namespace string) error {
	if len(releases) == 0 {
		printNoReleasesFound(namespace)
		return nil
	}

	switch format {
	case "json":
		return printJSON(releases)
	case "yaml":
		return printYAML(releases)
	default:
		printTable(releases)
	}
	return nil
}

func printNoReleasesFound(namespace string) {
	msg := pterm.DefaultParagraph.WithMaxWidth(80)
	if namespace == "" {
		msg.Println("No Helm releases found in any namespace")
	} else {
		msg.Printf("No Helm releases found in namespace %q\n", namespace)
	}
}

// List with JSON output
func printJSON(releases []*release.Release) error {
	data, err := json.MarshalIndent(convertToElements(releases), "", "  ")
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// List with YAML output
func printYAML(releases []*release.Release) error {
	data, err := yaml.Marshal(convertToElements(releases))
	if err != nil {
		return fmt.Errorf("yaml marshal error: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// List table
func printTable(releases []*release.Release) {
	table := pterm.TableData{{
		"NAME", "NAMESPACE", "REVISION", "UPDATED", "STATUS",
		"CHART", "APP VERSION",
	}}

	for _, r := range releases {
		table = append(table, []string{
			r.Name,
			r.Namespace,
			fmt.Sprintf("%d", r.Version),
			formatTime(r.Info.LastDeployed),
			r.Info.Status.String(),
			fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version),
			r.Chart.Metadata.AppVersion,
		})
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithData(table).
		WithBoxed().
		Render()
}

func convertToElements(releases []*release.Release) []map[string]interface{} {
	var elements []map[string]interface{}
	for _, r := range releases {
		elements = append(elements, map[string]interface{}{
			"name":        r.Name,
			"namespace":   r.Namespace,
			"revision":    r.Version,
			"updated":     formatTime(r.Info.LastDeployed),
			"status":      r.Info.Status.String(),
			"chart":       fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version),
			"app_version": r.Chart.Metadata.AppVersion,
		})
	}
	return elements
}

// time format
func formatTime(t helmtime.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
