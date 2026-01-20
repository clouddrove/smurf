package helm

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/clouddrove/smurf/internal/ai"
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
	useAI bool,
) error {
	// startTime := time.Now()

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
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("namespace creation failed: %w", err)
		}
	}

	// Initialize action config
	fmt.Printf("⚙️  Initializing Helm configuration...\n")
	actionConfig, err := initActionConfig(namespace, debug)
	if err != nil {
		printErrorSummary("failed to initialize helm", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("failed to initialize helm: %w", err)
	}

	// Verify release exists or not
	if err := verifyReleaseExists(actionConfig, releaseName, namespace, debug); err != nil {
		printErrorSummary("release verification failed", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
		return fmt.Errorf("release verification failed: %w", err)
	}

	// Load chart (supports repo + local)
	fmt.Printf("📊 Loading chart '%s'...\n", chartRef)
	chart, err := loadChart(chartRef, repoURL, version, debug)
	if err != nil {
		printErrorSummary("failed to load chart", releaseName, namespace, chartRef, err)
		ai.AIExplainError(useAI, err.Error())
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
		ai.AIExplainError(useAI, err.Error())
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

	// Check current pod status before upgrade
	fmt.Printf("📋 Checking current pod status before upgrade...\n")
	if err := printCurrentPodStatus(namespace, releaseName, debug, false); err != nil {
		pterm.Warning.Printf("Could not check current pod status: %v\n", err)
	}

	if debug {
		pterm.Printf("Upgrade client configured:\n")
		pterm.Printf("  - Namespace: %s\n", client.Namespace)
		pterm.Printf("  - Atomic: %t\n", client.Atomic)
		pterm.Printf("  - Timeout: %v\n", client.Timeout)
		pterm.Printf("  - History Max: %d\n", client.MaxHistory)
	}

	// Start upgrade in a goroutine so we can monitor simultaneously
	upgradeDone := make(chan error, 1)
	go func() {
		rel, err := client.Run(releaseName, chart, vals)
		if err != nil {
			printReleaseResources(namespace, releaseName)
			printErrorSummary("Helm upgradation", releaseName, namespace, chartRef, err)
			upgradeDone <- err
		} else {
			handleInstallationSuccess(rel, namespace)
			upgradeDone <- nil
		}
	}()

	// Wait for upgrade to complete
	select {
	case err := <-upgradeDone:
		if err != nil {
			// Print final pod status after failure
			fmt.Printf("\n🔍 Checking pod status after failed upgrade...\n")
			printCurrentPodStatus(namespace, releaseName, debug, false)

			printReleaseResources(namespace, releaseName)
			printErrorSummary("Helm upgrade failed", releaseName, namespace, chartRef, err)
			ai.AIExplainError(useAI, err.Error())

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
		// Upgrade succeeded
		fmt.Printf("\n✅ Upgrade completed successfully\n")

	case <-time.After(10 * time.Second):
		// Check status while upgrade is in progress
		fmt.Printf("\n⏳ Upgrade is taking longer than expected, checking status...\n")
		printCurrentPodStatus(namespace, releaseName, debug, false)
	}

	/*
		// Execute upgrade
		rel, err := client.Run(releaseName, chart, vals)
		if err != nil {
			// Print pod status after failed upgrade
			fmt.Printf("🔍 Checking pod status after failed upgrade...\n")
			printCurrentPodStatus(namespace, releaseName, debug, false)
			printReleaseResources(namespace, releaseName)
			printErrorSummary("Helm upgradation", releaseName, namespace, chartRef, err)
			ai.AIExplainError(useAI, err.Error())

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
		}*/

	// Verify readiness only if wait is enabled
	if wait {
		readinessTimeout := 5 * time.Minute
		if debug {
			pterm.Printf("Waiting for resources to be ready (timeout: %v)\n", readinessTimeout)
		}
		if err := verifyFinalReadiness(namespace, releaseName, readinessTimeout, debug); err != nil {
			ai.AIExplainError(useAI, err.Error())
			return fmt.Errorf("readiness verification failed: %w", err)
		}
	} else if debug {
		pterm.Println("Skipping readiness verification (wait=false)")
	}

	// duration := time.Since(startTime)
	// //pterm.Success.Printf("Release %q successfully upgraded in %s\n", rel.Name, duration)

	// if debug {
	// 	printReleaseInfo(rel, debug)
	// 	printResourcesFromRelease(rel)
	// 	pterm.Printf("=== HELM UPGRADE COMPLETED IN %s ===\n", duration)
	// }

	// handleInstallationSuccess(rel, namespace)
	return nil
}

// loadChart resolves both local and repo-based charts
// loadChart resolves both local, repo-based, and OCI charts
func loadChart(chartRef, repoURL, version string, debug bool) (*chart.Chart, error) {
	if debug {
		pterm.Printf("Resolving chart: %s\n", chartRef)
	}

	// Check for OCI registry reference FIRST
	if strings.HasPrefix(chartRef, "oci://") {
		fmt.Printf("🐳 Loading OCI chart from registry...\n")
		return LoadOCIChart(chartRef, version, cli.New(), debug)
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

// printCurrentPodStatus shows pods during upgrade and monitors for pending pods
func printCurrentPodStatus(namespace, releaseName string, debug bool, monitorChanges bool) error {
	clientset, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get kube client: %w", err)
	}

	// Get initial state of pods
	initialPods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Show current state
	showPodState("Current Pod State", initialPods.Items, releaseName, debug)

	// If monitoring changes, watch for new pods
	if monitorChanges {
		return watchForNewPods(clientset, namespace, releaseName, initialPods, debug)
	}

	return nil
}

// Watch for new pods during upgrade
func watchForNewPods(clientset *kubernetes.Clientset, namespace, releaseName string, initialPods *corev1.PodList, debug bool) error {
	fmt.Println("\n👀 Monitoring for new pods during upgrade...")

	seenPodNames := make(map[string]bool)
	for _, pod := range initialPods.Items {
		seenPodNames[pod.Name] = true
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second) // Monitor for 30 seconds
	newPodsDetected := false

	for {
		select {
		case <-ticker.C:
			currentPods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				continue // Skip this check if error
			}

			// Find new pods
			var newPods []corev1.Pod
			for _, pod := range currentPods.Items {
				if !seenPodNames[pod.Name] {
					newPods = append(newPods, pod)
					seenPodNames[pod.Name] = true
					newPodsDetected = true
				}
			}

			// Show new pods immediately
			if len(newPods) > 0 {
				fmt.Printf("\n🆕 New pods detected (%d):\n", len(newPods))
				showNewPodsDetails(newPods, debug)

				// Check if any new pod is from our release and is pending
				for _, pod := range newPods {
					if isPodFromRelease(pod, releaseName) {
						fmt.Printf("🎯 This pod belongs to release '%s'\n", releaseName)
						if pod.Status.Phase == corev1.PodPending {
							showPodStuckDetails(clientset, pod, namespace, debug)
						}
					}
				}
			}

		case <-timeout:
			if !newPodsDetected {
				fmt.Println("⏳ No new pods detected during monitoring period")
			}
			return nil
		}
	}
}

// Show detailed state of pods
// func showPodState(title string, pods []corev1.Pod, releaseName string, debug bool) {
// 	fmt.Printf("\n%s (%d pods):\n", title, len(pods))

// 	if len(pods) == 0 {
// 		fmt.Println("📭 No pods found")
// 		return
// 	}

// 	// Categorize pods
// 	var releasePods []corev1.Pod
// 	var otherPods []corev1.Pod
// 	statusCount := make(map[string]int)

// 	for _, pod := range pods {
// 		status := string(pod.Status.Phase)
// 		statusCount[status]++

// 		if isPodFromRelease(pod, releaseName) {
// 			releasePods = append(releasePods, pod)
// 		} else {
// 			otherPods = append(otherPods, pod)
// 		}
// 	}

// 	// Show summary
// 	printStatusSummary(statusCount, len(pods))

// 	// Show release pods
// 	if len(releasePods) > 0 {
// 		fmt.Printf("\n🎯 Pods for release '%s' (%d pods):\n", releaseName, len(releasePods))
// 		printPodTableDetailed(releasePods, debug)
// 	}

// 	// Show pending pods (all)
// 	var pendingPods []corev1.Pod
// 	for _, pod := range pods {
// 		if pod.Status.Phase == corev1.PodPending {
// 			pendingPods = append(pendingPods, pod)
// 		}
// 	}

// 	if len(pendingPods) > 0 {
// 		fmt.Printf("\n⏳ Pending Pods (%d):\n", len(pendingPods))
// 		for _, pod := range pendingPods {
// 			printQuickPodStatus(pod)
// 		}
// 	}

// 	// Show failed pods (all)
// 	var failedPods []corev1.Pod
// 	for _, pod := range pods {
// 		if pod.Status.Phase == corev1.PodFailed {
// 			failedPods = append(failedPods, pod)
// 		}
// 	}

// 	if len(failedPods) > 0 {
// 		fmt.Printf("\n❌ Failed Pods (%d):\n", len(failedPods))
// 		for _, pod := range failedPods {
// 			printQuickPodStatus(pod)
// 		}
// 	}
// }

// Show details of new pods
func showNewPodsDetails(newPods []corev1.Pod, debug bool) {
	for _, pod := range newPods {
		age := time.Since(pod.CreationTimestamp.Time).Round(time.Second)
		readyCount := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyCount++
			}
		}

		statusColor := pterm.FgGreen
		if pod.Status.Phase == corev1.PodPending {
			statusColor = pterm.FgYellow
		} else if pod.Status.Phase == corev1.PodFailed {
			statusColor = pterm.FgRed
		}

		statusColor.Printf("  %s: %s (Age: %s, Ready: %d/%d)\n",
			pod.Name,
			pod.Status.Phase,
			age,
			readyCount,
			len(pod.Spec.Containers))

		// Show immediate status
		if pod.Status.Phase == corev1.PodPending {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					pterm.Warning.Printf("    Container %s: %s\n", cs.Name, cs.State.Waiting.Reason)
				}
			}
		}
	}
}

// Show why pod is stuck
// func showPodStuckDetails(clientset *kubernetes.Clientset, pod corev1.Pod, namespace string, debug bool) {
// 	fmt.Println("\n🔍 Investigating pending pod:", pod.Name)

// 	// Get events immediately
// 	events, _ := getPodEvents(clientset, namespace, pod.Name)
// 	if len(events) > 0 {
// 		fmt.Println("  Recent Events:")
// 		for i, event := range events {
// 			if i >= 3 { // Show only 3 most recent events
// 				break
// 			}
// 			icon := "ℹ️"
// 			if event.Type == "Warning" {
// 				icon = "⚠️"
// 			}
// 			pterm.Printf("    %s [%s] %s: %s\n",
// 				icon,
// 				event.LastTimestamp.Format("15:04:05"),
// 				event.Reason,
// 				event.Message)
// 		}
// 	}

// 	// Check container status
// 	if len(pod.Status.ContainerStatuses) > 0 {
// 		fmt.Println("  Container Status:")
// 		for _, cs := range pod.Status.ContainerStatuses {
// 			if cs.State.Waiting != nil {
// 				pterm.Error.Printf("    %s: %s - %s\n",
// 					cs.Name,
// 					cs.State.Waiting.Reason,
// 					cs.State.Waiting.Message)
// 			}
// 		}
// 	}

// 	// Check conditions
// 	for _, cond := range pod.Status.Conditions {
// 		if cond.Status != corev1.ConditionTrue && cond.Message != "" {
// 			pterm.Warning.Printf("  Condition: %s - %s\n", cond.Reason, cond.Message)
// 		}
// 	}
// }

// Quick pod status
func printQuickPodStatus(pod corev1.Pod) {
	age := time.Since(pod.CreationTimestamp.Time).Round(time.Second)
	readyCount := 0
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			readyCount++
		}
	}

	statusIcon := "✅"
	if pod.Status.Phase == corev1.PodPending {
		statusIcon = "⏳"
	} else if pod.Status.Phase == corev1.PodFailed {
		statusIcon = "❌"
	} else if pod.Status.Phase == corev1.PodSucceeded {
		statusIcon = "✓"
	}

	// Get status message
	statusMsg := string(pod.Status.Phase)
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			statusMsg = cs.State.Waiting.Reason
			break
		}
	}

	fmt.Printf("  %s %s: %s (Age: %s, Ready: %d/%d, Restarts: %d)\n",
		statusIcon,
		pod.Name,
		statusMsg,
		age,
		readyCount,
		len(pod.Spec.Containers),
		getTotalRestarts(pod))
}

// Detailed table with all columns
func printPodTableDetailed(pods []corev1.Pod, debug bool) {
	if len(pods) == 0 {
		return
	}

	tableData := pterm.TableData{
		{"POD NAME", "STATUS", "READY", "RESTARTS", "AGE", "NODE", "MESSAGE"},
	}

	for _, pod := range pods {
		age := time.Since(pod.CreationTimestamp.Time).Round(time.Second)

		// Count ready containers
		readyContainers := 0
		totalContainers := len(pod.Spec.Containers)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
		}

		nodeName := pod.Spec.NodeName
		if nodeName == "" {
			nodeName = "<none>"
		}

		message := getPodStatusMessage(pod)

		tableData = append(tableData, []string{
			pod.Name,
			string(pod.Status.Phase),
			fmt.Sprintf("%d/%d", readyContainers, totalContainers),
			fmt.Sprintf("%d", getTotalRestarts(pod)),
			age.String(),
			nodeName,
			message,
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// Helper to determine if pod belongs to release - more flexible matching
func isPodFromRelease(pod corev1.Pod, releaseName string) bool {
	labels := pod.GetLabels()

	// Convert to lowercase for case-insensitive matching
	releaseNameLower := strings.ToLower(releaseName)
	podNameLower := strings.ToLower(pod.Name)

	// Strategy 1: Check app.kubernetes.io/instance label (standard Helm label)
	if instance, exists := labels["app.kubernetes.io/instance"]; exists {
		if strings.ToLower(instance) == releaseNameLower {
			return true
		}
	}

	// Strategy 2: Check for "release" label (common in older charts)
	if instance, exists := labels["release"]; exists {
		if strings.ToLower(instance) == releaseNameLower {
			return true
		}
	}

	// Strategy 3: Check if pod name contains release name
	if strings.Contains(podNameLower, releaseNameLower) {
		return true
	}

	// Strategy 4: Check for release in pod name pattern (common pattern: releaseName-*)
	pattern := regexp.MustCompile(fmt.Sprintf("^%s-[a-z0-9]+-[a-z0-9]+$", regexp.QuoteMeta(releaseNameLower)))
	if pattern.MatchString(podNameLower) {
		return true
	}

	// Strategy 5: Check for Helm-specific labels
	if heritage, exists := labels["heritage"]; exists && heritage == "Helm" {
		if instance, exists := labels["release"]; exists && strings.ToLower(instance) == releaseNameLower {
			return true
		}
	}

	return false
}

// Helper function to get pod logs
func getPodLogs(clientset *kubernetes.Clientset, namespace, podName, containerName string, tailLines int64) (string, error) {
	podLogOpts := corev1.PodLogOptions{
		Container: containerName,
		TailLines: &tailLines,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Print status summary
func printStatusSummary(statusCount map[string]int, total int) {
	fmt.Printf("📈 Status: ")

	// Create summary string
	parts := []string{}
	order := []string{"Pending", "Running", "Succeeded", "Failed", "Unknown"}
	icons := map[string]string{
		"Pending":   "⏳",
		"Running":   "✅",
		"Succeeded": "✓",
		"Failed":    "❌",
		"Unknown":   "❓",
	}

	for _, status := range order {
		if count, exists := statusCount[status]; exists && count > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", icons[status], count))
		}
	}

	if len(parts) > 0 {
		fmt.Println(strings.Join(parts, ", "))
	}
	fmt.Printf("📊 Total pods: %d\n", total)
}

// Helper function to get detailed pod status message
func getPodStatusMessage(pod corev1.Pod) string {
	switch pod.Status.Phase {
	case corev1.PodPending:
		// Check for specific pending reasons
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				return fmt.Sprintf("Container %s: %s", cs.Name, cs.State.Waiting.Reason)
			}
		}
		// Check pod conditions
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodScheduled && cond.Status != corev1.ConditionTrue {
				return fmt.Sprintf("Unscheduled: %s", cond.Reason)
			}
		}
		return "Scheduling"

	case corev1.PodRunning:
		// Check if all containers are ready
		allReady := true
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				allReady = false
				return fmt.Sprintf("Container %s not ready", cs.Name)
			}
		}
		if allReady {
			return "All containers ready"
		}

	case corev1.PodFailed:
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
				msg := fmt.Sprintf("Container %s failed", cs.Name)
				if cs.State.Terminated.Reason != "" {
					msg += fmt.Sprintf(": %s", cs.State.Terminated.Reason)
				}
				return msg
			}
		}
		return "Pod failed"

	case corev1.PodSucceeded:
		return "Completed successfully"

	case corev1.PodUnknown:
		return "Unknown state"
	}

	return string(pod.Status.Phase)
}

// Helper function to get pod events
func getPodEvents(clientset *kubernetes.Clientset, namespace, podName string) ([]corev1.Event, error) {
	events, err := clientset.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", podName),
	})
	if err != nil {
		return nil, err
	}

	// Sort by most recent
	sort.Slice(events.Items, func(i, j int) bool {
		return events.Items[i].LastTimestamp.After(events.Items[j].LastTimestamp.Time)
	})

	// Return last 5 events
	limit := 5
	if len(events.Items) < limit {
		limit = len(events.Items)
	}

	return events.Items[:limit], nil
}

// Helper function to get total restart count
func getTotalRestarts(pod corev1.Pod) int {
	total := 0
	for _, cs := range pod.Status.ContainerStatuses {
		total += int(cs.RestartCount)
	}
	return total
}

// ---------

// Show detailed pod information like kubectl describe
func describePod(clientset *kubernetes.Clientset, pod corev1.Pod, namespace string, debug bool) {
	fmt.Printf("\n📋 Pod Details: %s\n", pod.Name)
	fmt.Println(strings.Repeat("=", 50))

	// Status
	fmt.Println("\nStatus:")
	fmt.Printf("  Phase:   %s\n", pod.Status.Phase)
	fmt.Printf("  Reason:  %s\n", pod.Status.Reason)
	fmt.Printf("  Message: %s\n", pod.Status.Message)
	fmt.Printf("  Pod IP:  %s\n", pod.Status.PodIP)
	fmt.Printf("  Host IP: %s\n", pod.Status.HostIP)

	// Container Statuses
	if len(pod.Status.ContainerStatuses) > 0 {
		fmt.Println("\nContainers:")
		for i, cs := range pod.Status.ContainerStatuses {
			fmt.Printf("  Container %d: %s\n", i+1, cs.Name)
			fmt.Printf("    Container ID:  %s\n", cs.ContainerID)
			fmt.Printf("    Image:         %s\n", cs.Image)
			fmt.Printf("    Image ID:      %s\n", cs.ImageID)
			fmt.Printf("    Ready:         %v\n", cs.Ready)
			fmt.Printf("    Restart Count: %d\n", cs.RestartCount)

			// State
			if cs.State.Waiting != nil {
				fmt.Printf("    State:         Waiting\n")
				fmt.Printf("      Reason:      %s\n", cs.State.Waiting.Reason)
				fmt.Printf("      Message:     %s\n", cs.State.Waiting.Message)
			} else if cs.State.Running != nil {
				fmt.Printf("    State:         Running\n")
				fmt.Printf("      Started:     %s\n", cs.State.Running.StartedAt.Format("2006-01-02 15:04:05"))
			} else if cs.State.Terminated != nil {
				fmt.Printf("    State:         Terminated\n")
				fmt.Printf("      Exit Code:   %d\n", cs.State.Terminated.ExitCode)
				fmt.Printf("      Reason:      %s\n", cs.State.Terminated.Reason)
				fmt.Printf("      Message:     %s\n", cs.State.Terminated.Message)
				fmt.Printf("      Started:     %s\n", cs.State.Terminated.StartedAt.Format("2006-01-02 15:04:05"))
				fmt.Printf("      Finished:    %s\n", cs.State.Terminated.FinishedAt.Format("2006-01-02 15:04:05"))
			}

			// Last State (if any)
			if cs.LastTerminationState.Terminated != nil {
				fmt.Printf("    Last State:    Terminated\n")
				fmt.Printf("      Exit Code:   %d\n", cs.LastTerminationState.Terminated.ExitCode)
				fmt.Printf("      Reason:      %s\n", cs.LastTerminationState.Terminated.Reason)
			}
			fmt.Println()
		}
	}

	// Get recent events
	events, err := getPodEvents(clientset, namespace, pod.Name)
	if err == nil && len(events) > 0 {
		fmt.Println("\nEvents:")
		fmt.Println("  Type    Reason            Age   From               Message")
		fmt.Println("  ----    ------            ----  ----               -------")
		for _, event := range events {
			age := time.Since(event.LastTimestamp.Time).Round(time.Second)

			// Color coding for event type
			eventType := event.Type
			if event.Type == "Warning" {
				eventType = pterm.Red(event.Type)
			} else {
				eventType = pterm.Green(event.Type)
			}

			fmt.Printf("  %-7s %-17s %-5s %-18s %s\n",
				eventType,
				event.Reason,
				age.String(),
				event.Source.Component,
				event.Message)
		}
	}

	// Show logs for failed/error containers
	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodPending {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil || (cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0) {
				fmt.Printf("\n📝 Attempting to get logs for container '%s':\n", cs.Name)
				logs, err := getPodLogs(clientset, namespace, pod.Name, cs.Name, 20) // last 20 lines
				if err != nil {
					fmt.Printf("  ❌ Failed : %v\n", err)
				} else if logs != "" {
					fmt.Println("  Last 20 lines of logs:")
					fmt.Println(strings.Repeat("-", 40))
					fmt.Println(logs)
					fmt.Println(strings.Repeat("-", 40))
				}
			}
		}
	}

	fmt.Println(strings.Repeat("=", 50))
}

// Update the showPodStuckDetails to include describe
func showPodStuckDetails(clientset *kubernetes.Clientset, pod corev1.Pod, namespace string, debug bool) {
	fmt.Println("\n🔍 Investigating pending pod:", pod.Name)

	// Get events immediately
	events, _ := getPodEvents(clientset, namespace, pod.Name)
	if len(events) > 0 {
		fmt.Println("  Recent Events:")
		for i, event := range events {
			if i >= 3 { // Show only 3 most recent events
				break
			}
			icon := "ℹ️"
			if event.Type == "Warning" {
				icon = "⚠️"
			}
			pterm.Printf("    %s [%s] %s: %s\n",
				icon,
				event.LastTimestamp.Format("15:04:05"),
				event.Reason,
				event.Message)
		}
	}

	// Check container status
	if len(pod.Status.ContainerStatuses) > 0 {
		fmt.Println("  Container Status:")
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				pterm.Error.Printf("    %s: %s - %s\n",
					cs.Name,
					cs.State.Waiting.Reason,
					cs.State.Waiting.Message)
			}
		}
	}

	// Check conditions
	for _, cond := range pod.Status.Conditions {
		if cond.Status != corev1.ConditionTrue && cond.Message != "" {
			pterm.Warning.Printf("  Condition: %s - %s\n", cond.Reason, cond.Message)
		}
	}

	// Show detailed describe output for pending pods
	if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodFailed {
		describePod(clientset, pod, namespace, debug)
	}
}

// Update the printCurrentPodStatus to show describe for pending/failed pods
func showPodState(title string, pods []corev1.Pod, releaseName string, debug bool) {
	fmt.Printf("\n%s (%d pods):\n", title, len(pods))

	if len(pods) == 0 {
		fmt.Println("📭 No pods found")
		return
	}

	// Categorize pods
	var releasePods []corev1.Pod
	var otherPods []corev1.Pod
	var pendingPods []corev1.Pod
	var failedPods []corev1.Pod
	statusCount := make(map[string]int)

	for _, pod := range pods {
		status := string(pod.Status.Phase)
		statusCount[status]++

		if isPodFromRelease(pod, releaseName) {
			releasePods = append(releasePods, pod)
		} else {
			otherPods = append(otherPods, pod)
		}

		if pod.Status.Phase == corev1.PodPending {
			pendingPods = append(pendingPods, pod)
		} else if pod.Status.Phase == corev1.PodFailed {
			failedPods = append(failedPods, pod)
		}
	}

	// Show summary
	printStatusSummary(statusCount, len(pods))

	// Show release pods table
	if len(releasePods) > 0 {
		fmt.Printf("\n🎯 Pods for release '%s' (%d pods):\n", releaseName, len(releasePods))
		printPodTableDetailed(releasePods, debug)

		// Auto-describe pending/failed release pods
		clientset, _ := getKubeClient()
		for _, pod := range releasePods {
			if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodFailed {
				describePod(clientset, pod, pod.Namespace, debug)
			}
		}
	}

	// Show pending pods summary
	if len(pendingPods) > 0 {
		fmt.Printf("\n⏳ Pending Pods (%d):\n", len(pendingPods))
		for _, pod := range pendingPods {
			printQuickPodStatus(pod)
		}
	}

	// Show failed pods summary
	if len(failedPods) > 0 {
		fmt.Printf("\n❌ Failed Pods (%d):\n", len(failedPods))
		for _, pod := range failedPods {
			printQuickPodStatus(pod)
		}
	}
}
