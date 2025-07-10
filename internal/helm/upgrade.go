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
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HelmUpgrade performs a Helm upgrade operation for a specified release.
// It initializes the Helm action configuration, sets up the upgrade parameters,
// executes the upgrade, and then retrieves the status of the release post-upgrade.
// Detailed error logging is performed if any step fails.
func HelmUpgrade(releaseName, chartRef, namespace string, setValues []string, valuesFiles []string, setLiteral []string, createNamespace, atomic bool, timeout time.Duration, debug bool, repoURL string, version string) error {
	startTime := time.Now()
	logOperation(debug, "Starting Helm upgrade for release %s in namespace %s", releaseName, namespace)

	// Handle namespace creation separately since Upgrade doesn't have CreateNamespace
	// Handle namespace creation first
	if createNamespace {
		if err := ensureNamespace(namespace, true); err != nil {
			logDetailedError("namespace creation", err, namespace, releaseName)
			return err
		}
	}

	// Initialize action config once with proper namespace
	actionConfig, err := initActionConfig(namespace, debug)
	if err != nil {
		return fmt.Errorf("failed to initialize helm: %w", err)
	}

	// Verify the release exists in the target namespace
	if err := verifyReleaseExists(actionConfig, releaseName, namespace); err != nil {
		return fmt.Errorf("release verification failed: %w", err)
	}

	settings := cli.New()

	logFn := func(format string, v ...interface{}) {
		if debug {
			fmt.Printf(format, v...)
			fmt.Println()
		}
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logFn); err != nil {
		logDetailedError("helm action configuration", err, namespace, releaseName)
		return err
	}

	if actionConfig.KubeClient == nil {
		err := fmt.Errorf("KubeClient initialization failed")
		logDetailedError("kubeclient initialization", err, namespace, releaseName)
		return err
	}

	actionConfig, err = initActionConfig(namespace, debug)
	if err != nil {
		return fmt.Errorf("failed to initialize helm: %w", err)
	}

	chart, err := loadChart(chartRef, debug)
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteral, debug)
	if err != nil {
		return fmt.Errorf("failed to load values: %w", err)
	}

	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Atomic = atomic
	client.Timeout = timeout
	client.Wait = true
	client.WaitForJobs = true

	rel, err := client.Run(releaseName, chart, vals)
	if err != nil {
		pterm.Error.Printf("HELM UPGRADE FAILED: %v\n", err)

		// Get pod details for debugging
		pods, _ := getPods(namespace, releaseName)
		for _, pod := range pods {
			printPodDetails(pod) // Now matches the correct signature
		}

		return fmt.Errorf("upgrade failed: %w", err)
	}

	if err := verifyFinalReadiness(namespace, releaseName, 30*time.Second, debug); err != nil {
		return fmt.Errorf("readiness verification failed: %w", err)
	}

	pterm.Success.Printf("Release %q successfully upgraded in %s\n", rel.Name, time.Since(startTime))
	printReleaseInfo(rel, debug)
	printResourcesFromRelease(rel)
	return nil
}

func initActionConfig(namespace string, debug bool) (*action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace) // Explicitly set namespace

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(
		settings.RESTClientGetter(),
		namespace, // Pass namespace here
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			if debug {
				pterm.Debug.Printf(format, v...)
			}
		},
	)
	return actionConfig, err
}

func verifyReleaseExists(actionConfig *action.Configuration, releaseName, namespace string) error {
	listAction := action.NewList(actionConfig)
	listAction.AllNamespaces = false
	listAction.All = true
	listAction.SetStateMask()

	releases, err := listAction.Run()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	for _, r := range releases {
		if r.Name == releaseName && r.Namespace == namespace {
			return nil
		}
	}

	return fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
}

// Load chart from path or repo
func loadChart(chartRef string, debug bool) (*chart.Chart, error) {
	logOperation(debug, "Resolving chart path: %s", chartRef)
	absPath, err := filepath.Abs(chartRef)
	if err != nil {
		logOperation(debug, "Path resolution failed: %v", err)
		return nil, fmt.Errorf("failed to resolve chart path: %w", err)
	}
	logOperation(debug, "Absolute chart path: %s", absPath)

	chart, err := loader.Load(absPath)
	if err != nil {
		logOperation(debug, "Chart loading failed: %v", err)
		return nil, err
	}
	logOperation(debug, "Chart metadata loaded - Name: %s, Version: %s",
		chart.Metadata.Name, chart.Metadata.Version)
	return chart, nil
}

// loadAndMergeValuesWithSets loads values from the specified files and merges them with the set values.
// It returns the merged values map or an error if the values cannot be loaded or parsed.
// The set values are applied after loading the values from the files.
// loadAndMergeValuesWithSets loads values from the specified files and merges them with the set values.
// It properly handles relative paths for nested values files.
func loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteralValues []string, debug bool) (map[string]interface{}, error) {
	logOperation(debug, "Starting values files processing")
	resolvedFiles, err := resolveValuesPaths(valuesFiles, debug)
	if err != nil {
		return nil, err
	}

	vals := make(map[string]interface{})
	for i, f := range resolvedFiles {
		logOperation(debug, "Processing values file %d/%d: %s", i+1, len(resolvedFiles), f)
		currentVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			logOperation(debug, "Error reading values file: %v", err)
			return nil, fmt.Errorf("failed to read %s: %w", f, err)
		}
		vals = mergeMaps(vals, currentVals)
	}

	for _, set := range setValues {
		logOperation(debug, "Applying --set value: %s", set)
		if err := strvals.ParseInto(set, vals); err != nil {
			logOperation(debug, "Error parsing --set value: %v", err)
			return nil, fmt.Errorf("invalid --set value %s: %w", set, err)
		}
	}

	for _, setLiteral := range setLiteralValues {
		logOperation(debug, "Applying --set-literal value: %s", setLiteral)
		if err := strvals.ParseIntoString(setLiteral, vals); err != nil {
			logOperation(debug, "Error parsing --set-literal value: %v", err)
			return nil, fmt.Errorf("invalid --set-literal value %s: %w", setLiteral, err)
		}
	}

	logOperation(debug, "Successfully merged all values")
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
func verifyFinalReadiness(namespace, releaseName string, timeout time.Duration, debug bool) error {
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
			logOperation(debug, "All pods are ready")
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}

// Helper function to print pod details
// printPodDetails now matches all call sites
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
func resolveValuesPaths(valuesFiles []string, debug bool) ([]string, error) {
	var resolved []string
	for i, f := range valuesFiles {
		logOperation(debug, "Resolving path for values file %d: %s", i+1, f)
		absPath, err := filepath.Abs(f)
		if err != nil {
			logOperation(debug, "Path resolution failed: %v", err)
			return nil, fmt.Errorf("failed to resolve path %s: %w", f, err)
		}

		if _, err := os.Stat(absPath); err != nil {
			logOperation(debug, "Values file not found: %v", err)
			return nil, fmt.Errorf("values file not found: %s", absPath)
		}

		resolved = append(resolved, absPath)
		logOperation(debug, "Resolved path: %s -> %s", f, absPath)
	}
	return resolved, nil
}
