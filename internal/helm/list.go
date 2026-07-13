package helm

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/clouddrove/smurf/internal/ai"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
)

// ListReleases returns and displays Helm releases with flexible output formats
func ListReleases(namespace, format string, useAI bool) ([]*release.Release, error) {
	cfg := new(action.Configuration)
	if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		ai.AIExplainError(useAI, err.Error())
		return nil, fmt.Errorf("helm init failed: %w", err)
	}

	client := action.NewList(cfg)
	client.AllNamespaces = namespace == ""
	client.StateMask = action.ListAll // Show all statuses

	releases, err := client.Run()
	if err != nil {
		ai.AIExplainError(useAI, err.Error())
		return nil, fmt.Errorf("release listing failed: %w", err)
	}

	if err := printOutput(releases, format, namespace); err != nil {

		return releases, err // Return releases even if printing fails
	}

	return releases, nil
}

// ListReleaseNames returns just the names of Helm releases in the given
// namespace (all namespaces if empty), for use in shell completion. Unlike
// ListReleases it never prints anything and never blocks longer than
// timeout: the Helm/Kubernetes call runs in the background and, if it has
// not finished by the deadline, a timeout error is returned immediately so a
// slow or unreachable cluster can't hang shell completion.
func ListReleaseNames(namespace string, timeout time.Duration) ([]string, error) {
	type result struct {
		names []string
		err   error
	}
	done := make(chan result, 1)

	go func() {
		cfg := new(action.Configuration)
		if err := cfg.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), func(string, ...interface{}) {}); err != nil {
			done <- result{nil, err}
			return
		}

		client := action.NewList(cfg)
		client.AllNamespaces = namespace == ""
		client.StateMask = action.ListAll

		releases, err := client.Run()
		if err != nil {
			done <- result{nil, err}
			return
		}

		names := make([]string, 0, len(releases))
		for _, r := range releases {
			names = append(names, r.Name)
		}
		done <- result{names, nil}
	}()

	select {
	case res := <-done:
		return res.names, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out after %s listing helm releases", timeout)
	}
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
