package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HelmUpgrade performs a Helm upgrade operation for a specified release.
// It initializes the Helm action configuration, sets up the upgrade parameters,
// executes the upgrade, and then retrieves the status of the release post-upgrade.
// Detailed error logging is performed if any step fails.
func HelmUpgrade(releaseName, chartRef, namespace string, setValues []string, valuesFiles []string, setLiteral []string, createNamespace, atomic bool, timeout time.Duration, debug bool, repoURL string, version string) error {
	// Initialize action config
	actionConfig, err := initActionConfig(namespace, debug)
	if err != nil {
		return fmt.Errorf("failed to initialize helm: %w", err)
	}

	// Load chart
	chart, err := loadChart(chartRef)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	// Load and merge values
	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues, nil)
	if err != nil {
		return fmt.Errorf("failed to load values: %w", err)
	}

	// Create upgrade client
	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Atomic = atomic
	client.Timeout = timeout
	client.Wait = true
	client.WaitForJobs = true

	// Run upgrade
	rel, err := client.Run(releaseName, chart, vals)
	if err != nil {
		// Enhanced error reporting
		pods, _ := getPods(namespace, releaseName)
		for _, pod := range pods {
			printPodDetails(pod)
		}
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Verify final readiness
	if err := verifyFinalReadiness(namespace, releaseName, 30*time.Second); err != nil {
		return fmt.Errorf("upgrade completed but readiness check failed: %w", err)
	}

	pterm.Success.Printf("Release %q successfully upgraded\n", rel.Name)
	return nil
}

// Initialize Helm action configuration
func initActionConfig(namespace string, debug bool) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(
		settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			if debug {
				pterm.Debug.Printf(format, v...)
			}
		},
	)
	return actionConfig, err
}

// Load chart from path or repo
func loadChart(chartRef string) (*chart.Chart, error) {
	// Handle local charts
	absPath, err := filepath.Abs(chartRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve chart path: %w", err)
	}
	return loader.Load(absPath)
}

// loadAndMergeValuesWithSets loads values from the specified files and merges them with the set values.
// It returns the merged values map or an error if the values cannot be loaded or parsed.
// The set values are applied after loading the values from the files.
// loadAndMergeValuesWithSets loads values from the specified files and merges them with the set values.
// It properly handles relative paths for nested values files.
func loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteralValues []string) (map[string]interface{}, error) {
	resolvedFiles, err := resolveValuesPaths(valuesFiles)
	if err != nil {
		return nil, err
	}

	vals := make(map[string]interface{})
	for _, f := range resolvedFiles {
		currentVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", f, err)
		}
		vals = mergeMaps(vals, currentVals)
	}

	for _, set := range setValues {
		if err := strvals.ParseInto(set, vals); err != nil {
			return nil, fmt.Errorf("invalid --set value %s: %w", set, err)
		}
	}

	for _, setLiteral := range setLiteralValues {
		if err := strvals.ParseIntoString(setLiteral, vals); err != nil {
			return nil, fmt.Errorf("invalid --set-literal value %s: %w", setLiteral, err)
		}
	}

	return vals, nil
}

// Get pods for a release
func getPods(namespace, releaseName string) ([]corev1.Pod, error) {
	clientset, err := getKubeClient()
	if err != nil {
		return nil, err
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// Verify final pod readiness
func verifyFinalReadiness(namespace, releaseName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("readiness verification timed out after %s", timeout)
		}

		pods, err := getPods(namespace, releaseName)
		if err != nil {
			return fmt.Errorf("failed to get pods: %w", err)
		}

		allReady := true
		for _, pod := range pods {
			if pod.Status.Phase != corev1.PodRunning {
				allReady = false
				break
			}
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status != corev1.ConditionTrue {
					allReady = false
					break
				}
			}
		}

		if allReady {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}

// Helper function to print pod details
func printPodDetails(pod corev1.Pod) {
	pterm.Info.Println("Pod:", pod.Name, "Status:", pod.Status.Phase)
	for _, cond := range pod.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			pterm.Error.Println("  Condition:", cond.Type, "Reason:", cond.Reason, "Message:", cond.Message)
		}
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			pterm.Warning.Println("  Container not ready:", cs.Name)
			if cs.State.Waiting != nil {
				pterm.Println("    Reason:", cs.State.Waiting.Reason)
				pterm.Println("    Message:", cs.State.Waiting.Message)
			}
		}
	}
}

func resolveValuesPaths(valuesFiles []string) ([]string, error) {
	var resolved []string
	for _, f := range valuesFiles {
		absPath, err := filepath.Abs(f)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", f, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return nil, fmt.Errorf("values file not found: %s", absPath)
		}
		resolved = append(resolved, absPath)
	}
	return resolved, nil
}
