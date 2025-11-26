package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	"k8s.io/client-go/kubernetes"
)

// HelmUpgrade performs a Helm upgrade operation with repo + local support
func HelmUpgrade(
	releaseName, chartRef, namespace string,
	setValues []string, valuesFiles []string, setLiteral []string,
	createNamespace, atomic bool,
	timeout time.Duration, debug bool,
	repoURL string, version string,
	wait bool,
	historyMax int,
) error {
	startTime := time.Now()

	if debug {
		pterm.Println("=== HELM UPGRADE STARTED ===")
		pterm.Printf("Release: %s\n", releaseName)
		pterm.Printf("Chart: %s\n", chartRef)
		pterm.Printf("Namespace: %s\n", namespace)
		pterm.Printf("Create Namespace: %t\n", createNamespace)
		pterm.Printf("Atomic: %t\n", atomic)
		pterm.Printf("Timeout: %v\n", timeout)
		pterm.Printf("Wait: %t\n", wait)
		pterm.Printf("Set values: %v\n", setValues)
		pterm.Printf("Values files: %v\n", valuesFiles)
		pterm.Printf("Set literal: %v\n", setLiteral)
		pterm.Printf("Repo URL: %s\n", repoURL)
		pterm.Printf("Version: %s\n", version)
		pterm.Printf("History Max: %d\n", historyMax)
	}

	// Handle namespace creation
	fmt.Printf("📦 Ensuring namespace '%s' exists...\n", namespace)
	if createNamespace {
		if debug {
			pterm.Println("Creating namespace if not exists...")
		}
		if err := ensureNamespace(namespace, debug); err != nil {
			printErrorSummary("namespace creation failed", releaseName, namespace, chartRef, err)
			return fmt.Errorf("namespace creation failed: %w", err)
		}
	}

	// Initialize action config
	fmt.Printf("⚙️  Initializing Helm configuration...\n")
	actionConfig, err := initActionConfig(namespace, debug)
	if err != nil {
		printErrorSummary("failed to initialize helm", releaseName, namespace, chartRef, err)
		return fmt.Errorf("failed to initialize helm: %w", err)
	}

	// Verify release exists or not
	if err := verifyReleaseExists(actionConfig, releaseName, namespace, debug); err != nil {
		printErrorSummary("release verification failed", releaseName, namespace, chartRef, err)
		return fmt.Errorf("release verification failed: %w", err)
	}

	// Load chart (supports repo + local)
	fmt.Printf("📊 Loading chart '%s'...\n", chartRef)
	chart, err := loadChart(chartRef, repoURL, version, debug)
	if err != nil {
		printErrorSummary("failed to load chart", releaseName, namespace, chartRef, err)
		return fmt.Errorf("failed to load chart: %w", err)
	}
	if debug {
		pterm.Printf("Chart loaded: %s (version %s)\n", chart.Name(), chart.Metadata.Version)
	}

	// Load and merge values
	fmt.Printf("📝 Processing values and configurations...\n")
	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteral, debug)
	if err != nil {
		printErrorSummary("failed to load values", releaseName, namespace, chartRef, err)
		return fmt.Errorf("failed to load values: %w", err)
	}

	// Create upgrade client
	fmt.Printf("🛠️  Setting up upgrade action...\n")
	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Atomic = atomic
	client.Timeout = timeout
	client.Wait = wait
	client.WaitForJobs = wait
	client.MaxHistory = historyMax

	if debug {
		pterm.Printf("Upgrade client configured:\n")
		pterm.Printf("  - Namespace: %s\n", client.Namespace)
		pterm.Printf("  - Atomic: %t\n", client.Atomic)
		pterm.Printf("  - Timeout: %v\n", client.Timeout)
		pterm.Printf("  - History Max: %d\n", client.MaxHistory)
	}

	// Execute upgrade
	rel, err := client.Run(releaseName, chart, vals)
	if err != nil {
		printReleaseResources(namespace, releaseName)
		printErrorSummary("Helm upgradation", releaseName, namespace, chartRef, err)

		// Debug pod info if upgrade fails
		if debug {
			pods, err := getPods(namespace, releaseName)
			if err == nil {
				pterm.Printf("Found %d pods for release %s\n", len(pods), releaseName)
				for _, pod := range pods {
					printPodDetails(pod)
				}
			}
		}
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Verify readiness only if wait is enabled
	if wait {
		readinessTimeout := 5 * time.Minute
		if debug {
			pterm.Printf("Waiting for resources to be ready (timeout: %v)\n", readinessTimeout)
		}
		if err := verifyFinalReadiness(namespace, releaseName, readinessTimeout, debug); err != nil {
			return fmt.Errorf("readiness verification failed: %w", err)
		}
	} else if debug {
		pterm.Println("Skipping readiness verification (wait=false)")
	}

	duration := time.Since(startTime)
	//pterm.Success.Printf("Release %q successfully upgraded in %s\n", rel.Name, duration)

	if debug {
		printReleaseInfo(rel, debug)
		printResourcesFromRelease(rel)
		pterm.Printf("=== HELM UPGRADE COMPLETED IN %s ===\n", duration)
	}

	handleInstallationSuccess(rel, namespace)
	return nil
}

// loadChart resolves both local and repo-based charts
func loadChart(chartRef, repoURL, version string, debug bool) (*chart.Chart, error) {
	if debug {
		pterm.Printf("Resolving chart: %s\n", chartRef)
	}

	// Local path (./chart or /path/to/chart)
	if strings.HasPrefix(chartRef, "./") || strings.HasPrefix(chartRef, "/") {
		absPath, err := filepath.Abs(chartRef)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve chart path: %w", err)
		}
		if debug {
			pterm.Printf("Loading local chart from: %s\n", absPath)
		}
		return loader.Load(absPath)
	}

	// Repo chart (repo/chart)
	settings := cli.New()
	chartPathOptions := action.ChartPathOptions{
		RepoURL: repoURL,
		Version: version,
	}

	if debug {
		pterm.Printf("Fetching chart %s from repo (version=%s, repo=%s)\n", chartRef, version, repoURL)
	}
	cp, err := chartPathOptions.LocateChart(chartRef, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart %s: %w", chartRef, err)
	}

	if debug {
		pterm.Printf("Chart resolved to local path: %s\n", cp)
	}

	return loader.Load(cp)
}

func initActionConfig(namespace string, debug bool) (*action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)

	logFn := func(format string, v ...interface{}) {
		if debug {
			message := fmt.Sprintf(format, v...)
			pterm.Printfln("HELM-CLI: %s", strings.TrimSpace(message))
		}
	}

	err := actionConfig.Init(
		settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"),
		logFn,
	)

	if debug && err == nil {
		pterm.Printf("Action config initialized for namespace: %s\n", namespace)
	}

	return actionConfig, err
}

func verifyReleaseExists(actionConfig *action.Configuration, releaseName, namespace string, debug bool) error {
	if debug {
		pterm.Printf("Checking if release %s exists in namespace %s\n", releaseName, namespace)
	}

	listAction := action.NewList(actionConfig)
	listAction.AllNamespaces = false
	listAction.All = true
	listAction.SetStateMask()

	releases, err := listAction.Run()
	if err != nil {
		if debug {
			pterm.Printf("Failed to list releases: %v\n", err)
		}
		return fmt.Errorf("failed to list releases: %w", err)
	}

	if debug {
		pterm.Printf("Found %d releases total\n", len(releases))
	}

	found := false
	for _, r := range releases {
		if r.Name == releaseName && r.Namespace == namespace {
			found = true
			if debug {
				pterm.Printf("Release found: %s (status: %s, version: %d)\n",
					releaseName, r.Info.Status, r.Version)
			}
			break
		}
	}

	if !found {
		if debug {
			pterm.Printf("Release %s not found in namespace %s\n", releaseName, namespace)
			pterm.Printf("Available releases in namespace %s:\n", namespace)
			for _, r := range releases {
				if r.Namespace == namespace {
					pterm.Printf("  - %s (status: %s)\n", r.Name, r.Info.Status)
				}
			}
		}
		return fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
	}

	return nil
}

func loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteralValues []string, debug bool) (map[string]interface{}, error) {
	if debug {
		pterm.Printf("Loading values from %d files\n", len(valuesFiles))
		pterm.Printf("Applying %d set values\n", len(setValues))
		pterm.Printf("Applying %d literal values\n", len(setLiteralValues))
	}

	resolvedFiles, err := resolveValuesPaths(valuesFiles, debug)
	if err != nil {
		return nil, err
	}

	vals := make(map[string]interface{})
	for i, f := range resolvedFiles {
		if debug {
			pterm.Printf("Reading values file %d: %s\n", i+1, f)
		}
		currentVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			if debug {
				pterm.Printf("Error reading values file: %v\n", err)
			}
			return nil, fmt.Errorf("failed to read %s: %w", f, err)
		}
		vals = mergeMaps(vals, currentVals)
	}

	for i, set := range setValues {
		if debug {
			pterm.Printf("Applying set value %d: %s\n", i+1, set)
		}
		if err := strvals.ParseInto(set, vals); err != nil {
			if debug {
				pterm.Printf("Error parsing set value: %v\n", err)
			}
			return nil, fmt.Errorf("invalid --set value %s: %w", set, err)
		}
	}

	for i, setLiteral := range setLiteralValues {
		if debug {
			pterm.Printf("Applying literal value %d: %s\n", i+1, setLiteral)
		}
		if err := strvals.ParseIntoString(setLiteral, vals); err != nil {
			if debug {
				pterm.Printf("Error parsing literal value: %v\n", err)
			}
			return nil, fmt.Errorf("invalid --set-literal value %s: %w", setLiteral, err)
		}
	}

	if debug {
		pterm.Println("All values processed successfully")
	}

	return vals, nil
}

func resolveValuesPaths(valuesFiles []string, debug bool) ([]string, error) {
	var resolved []string
	for i, f := range valuesFiles {
		if debug {
			pterm.Printf("Resolving values file %d: %s\n", i+1, f)
		}
		absPath, err := filepath.Abs(f)
		if err != nil {
			if debug {
				pterm.Printf("Path resolution failed: %v\n", err)
			}
			return nil, fmt.Errorf("failed to resolve path %s: %w", f, err)
		}

		if _, err := os.Stat(absPath); err != nil {
			if debug {
				pterm.Printf("Values file not found: %v\n", err)
			}
			return nil, fmt.Errorf("values file not found: %s", absPath)
		}

		resolved = append(resolved, absPath)
		if debug {
			pterm.Printf("Resolved to: %s\n", absPath)
		}
	}
	return resolved, nil
}

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

func verifyFinalReadiness(namespace, releaseName string, timeout time.Duration, debug bool) error {
	if debug {
		pterm.Printf("Verifying readiness with timeout: %v\n", timeout)
	}

	deadline := time.Now().Add(timeout)
	pollInterval := 5 * time.Second // Increased from 2 to 5 seconds

	clientset, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get kube client: %w", err)
	}

	for attempt := 1; ; attempt++ {
		if time.Now().After(deadline) {
			// Provide detailed timeout information
			pods, _ := getPods(namespace, releaseName)
			return fmt.Errorf("readiness verification timed out after %s. %d pods found. Check pod logs for details",
				timeout, len(pods))
		}

		if debug {
			pterm.Printf("Readiness check attempt %d\n", attempt)
		}

		// Check deployments, statefulsets, daemonsets first
		allWorkloadsReady, workloadStatus, err := checkWorkloadReadiness(clientset, namespace, releaseName, debug)
		if err != nil {
			if debug {
				pterm.Printf("Error checking workloads: %v\n", err)
			}
			continue // Retry on API errors
		}

		if !allWorkloadsReady {
			if debug {
				pterm.Printf("Workloads not ready: %s\n", workloadStatus)
			}
			time.Sleep(pollInterval)
			continue
		}

		// Then check pods
		pods, err := getPods(namespace, releaseName)
		if err != nil {
			if debug {
				pterm.Printf("Failed to get pods: %v\n", err)
			}
			time.Sleep(pollInterval)
			continue
		}

		if len(pods) == 0 {
			if debug {
				pterm.Printf("No pods found for release %s\n", releaseName)
			}
			time.Sleep(pollInterval)
			continue
		}

		allReady, notReadyPods := checkPodReadiness(pods, debug)
		if allReady {
			if debug {
				pterm.Println("All pods and workloads are ready")
			}
			return nil
		}

		if debug {
			pterm.Printf("Pods not ready: %v\n", notReadyPods)
			// Print pod details for debugging
			for _, pod := range pods {
				if !isPodReady(pod) {
					printPodDetails(pod)
				}
			}
		}

		time.Sleep(pollInterval)
	}
}

func checkWorkloadReadiness(clientset *kubernetes.Clientset, namespace, releaseName string, debug bool) (bool, string, error) {
	labelSelector := fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName)

	// Check Deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return false, "", err
	}

	for _, dep := range deployments.Items {
		if dep.Status.ReadyReplicas != *dep.Spec.Replicas {
			status := fmt.Sprintf("Deployment/%s: %d/%d ready",
				dep.Name, dep.Status.ReadyReplicas, *dep.Spec.Replicas)
			return false, status, nil
		}
	}

	// Check StatefulSets
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return false, "", err
	}

	for _, ss := range statefulsets.Items {
		if ss.Status.ReadyReplicas != *ss.Spec.Replicas {
			status := fmt.Sprintf("StatefulSet/%s: %d/%d ready",
				ss.Name, ss.Status.ReadyReplicas, *ss.Spec.Replicas)
			return false, status, nil
		}
	}

	// Check DaemonSets
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return false, "", err
	}

	for _, ds := range daemonsets.Items {
		if ds.Status.NumberReady != ds.Status.DesiredNumberScheduled {
			status := fmt.Sprintf("DaemonSet/%s: %d/%d ready",
				ds.Name, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
			return false, status, nil
		}
	}

	return true, "all workloads ready", nil
}

func checkPodReadiness(pods []corev1.Pod, debug bool) (bool, []string) {
	var notReadyPods []string
	allReady := true

	for _, pod := range pods {
		if !isPodReady(pod) {
			allReady = false
			status := fmt.Sprintf("Pod/%s: %s", pod.Name, pod.Status.Phase)
			if pod.Status.Phase == corev1.PodRunning {
				// If running but not ready, check container statuses
				for _, cs := range pod.Status.ContainerStatuses {
					if !cs.Ready {
						status = fmt.Sprintf("Pod/%s: container %s not ready", pod.Name, cs.Name)
						break
					}
				}
			}
			notReadyPods = append(notReadyPods, status)
		}
	}

	return allReady, notReadyPods
}

func isPodReady(pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
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
