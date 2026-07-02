package helm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const upgradePodPollInterval = 2 * time.Second

type failedPodSnapshot struct {
	pod   *corev1.Pod
	cache *cachedPodDiagnostics
}

// upgradePodMonitor watches pods during a Helm upgrade and snapshots unhealthy states
// plus logs/events before atomic rollback removes pods from the cluster.
type upgradePodMonitor struct {
	clientset   *kubernetes.Clientset
	namespace   string
	releaseName string
	debug       bool

	mu          sync.Mutex
	snapshots   map[string]*corev1.Pod
	diagnostics map[string]*cachedPodDiagnostics
	seenPods    map[string]bool
	stopCh      chan struct{}
	doneCh      chan struct{}
}

func newUpgradePodMonitor(namespace, releaseName string, debug bool) (*upgradePodMonitor, error) {
	clientset, err := getKubeClient()
	if err != nil {
		return nil, err
	}

	return &upgradePodMonitor{
		clientset:   clientset,
		namespace:   namespace,
		releaseName: releaseName,
		debug:       debug,
		snapshots:   make(map[string]*corev1.Pod),
		diagnostics: make(map[string]*cachedPodDiagnostics),
		seenPods:    make(map[string]bool),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}, nil
}

func (m *upgradePodMonitor) start() {
	go func() {
		defer close(m.doneCh)

		ticker := time.NewTicker(upgradePodPollInterval)
		defer ticker.Stop()

		m.poll()
		for {
			select {
			case <-m.stopCh:
				m.poll()
				return
			case <-ticker.C:
				m.poll()
			}
		}
	}()
}

func (m *upgradePodMonitor) stop() {
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
	<-m.doneCh
}

func (m *upgradePodMonitor) poll() {
	pods, err := getPods(m.namespace, m.releaseName)
	if err != nil {
		if m.debug {
			pterm.Debug.Printf("upgrade monitor: failed to list pods: %v\n", err)
		}
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	currentPods := make(map[string]bool, len(pods))
	for i := range pods {
		pod := pods[i]
		currentPods[pod.Name] = true
		m.seenPods[pod.Name] = true

		if isPodUnhealthyForUpgrade(&pod) {
			m.snapshots[pod.Name] = pod.DeepCopy()
			fresh := capturePodDiagnostics(m.clientset, m.namespace, &pod)
			m.diagnostics[pod.Name] = mergePodDiagnostics(m.diagnostics[pod.Name], fresh)

			if m.debug {
				pterm.Debug.Printf("upgrade monitor: captured diagnostics for unhealthy pod %s\n", pod.Name)
			}
		}
	}

	// Pods that disappeared (scaled down / rollback) — mark cache as pod removed
	for name := range m.seenPods {
		if currentPods[name] {
			continue
		}
		if cache, ok := m.diagnostics[name]; ok {
			cache.podExists = false
		}
		if m.debug {
			pterm.Debug.Printf("upgrade monitor: pod %s removed from cluster\n", name)
		}
	}
}

func (m *upgradePodMonitor) failedSnapshots() []failedPodSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]failedPodSnapshot, 0, len(m.snapshots))
	for name, pod := range m.snapshots {
		cache := m.diagnostics[name]
		if cache == nil {
			cache = newCachedPodDiagnostics()
			cache.podExists = false
		}
		result = append(result, failedPodSnapshot{
			pod:   pod.DeepCopy(),
			cache: cache,
		})
	}
	return result
}

func isPodUnhealthyForUpgrade(pod *corev1.Pod) bool {
	if isPodInFailureState(pod) {
		return true
	}

	if pod.Status.Phase == corev1.PodFailed {
		return true
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
			return true
		}
		if cs.State.Waiting != nil {
			switch cs.State.Waiting.Reason {
			case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull",
				"CreateContainerConfigError", "InvalidImageName", "CreateContainerError":
				return true
			}
		}
	}

	if pod.Status.Phase == corev1.PodRunning && !isPodReady(*pod) {
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready && cs.State.Waiting != nil {
				return true
			}
		}
	}

	return false
}

func printDiagnosticsBanner(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
	pterm.Error.Println(title)
	fmt.Println(strings.Repeat("═", 80))
}

func printDiagnosticsSubSection(title string) {
	fmt.Println()
	pterm.DefaultSection.WithLevel(2).Println(title)
	fmt.Println(strings.Repeat("─", 80))
}

func printFailedPodReport(clientset *kubernetes.Clientset, namespace string, snapshot failedPodSnapshot, index, total int) {
	pod := *snapshot.pod
	status := getKubectlLikeStatus(pod)

	printDiagnosticsSubSection(fmt.Sprintf("Failed Pod [%d/%d]: %s", index, total, pod.Name))
	fmt.Printf("  Namespace : %s\n", namespace)
	fmt.Printf("  Status    : %s\n", status)
	fmt.Printf("  Phase     : %s\n", pod.Status.Phase)
	fmt.Printf("  Node      : %s\n", podOrNone(pod.Spec.NodeName))

	if snapshot.cache != nil && !snapshot.cache.capturedAt.IsZero() {
		fmt.Printf("  Captured  : %s\n", snapshot.cache.capturedAt.Format(dateTimeFormat))
	}
	if snapshot.cache != nil && !snapshot.cache.podExists {
		pterm.Warning.Println("  Note      : Pod was removed during rollback — showing captured state")
	}

	for _, cs := range pod.Status.ContainerStatuses {
		state := containerStateSummary(cs)
		if state != "" {
			fmt.Printf("  Container : %s — %s\n", cs.Name, state)
		}
	}

	describePodFromSnapshot(clientset, pod, namespace, snapshot.cache)
	printFailedPodLogsWithCache(clientset, namespace, pod, snapshot.cache)
}

func describePodFromSnapshot(clientset *kubernetes.Clientset, pod corev1.Pod, namespace string, cache *cachedPodDiagnostics) {
	fmt.Printf("\n📋 Pod Details: %s\n", pod.Name)
	fmt.Println(strings.Repeat("=", 50))

	fmt.Println("\nStatus:")
	fmt.Printf("  Phase:   %s\n", pod.Status.Phase)
	fmt.Printf("  Reason:  %s\n", pod.Status.Reason)
	fmt.Printf("  Message: %s\n", pod.Status.Message)

	if len(pod.Status.ContainerStatuses) > 0 {
		fmt.Println("\nContainers:")
		for i, cs := range pod.Status.ContainerStatuses {
			fmt.Printf("  Container %d: %s\n", i+1, cs.Name)
			fmt.Printf("    Image:         %s\n", cs.Image)
			fmt.Printf("    Ready:         %v\n", cs.Ready)
			if cs.State.Waiting != nil {
				fmt.Printf("    State:         Waiting (%s)\n", cs.State.Waiting.Reason)
				fmt.Printf("    Message:       %s\n", cs.State.Waiting.Message)
			}
		}
	}

	if cache != nil && len(cache.events) > 0 {
		printPodEventsSection(cache.events, "Events (captured during failure)")
		return
	}

	events, err := getPodEvents(clientset, namespace, pod.Name)
	if err == nil && len(events) > 0 {
		printPodEventsSection(events, "Events")
	}

	fmt.Println(strings.Repeat("=", 50))
}

func podOrNone(value string) string {
	if value == "" {
		return none
	}
	return value
}

func containerStateSummary(cs corev1.ContainerStatus) string {
	switch {
	case cs.State.Waiting != nil:
		return fmt.Sprintf("Waiting (%s): %s", cs.State.Waiting.Reason, cs.State.Waiting.Message)
	case cs.State.Terminated != nil:
		return fmt.Sprintf("Terminated (%s, exit %d): %s",
			cs.State.Terminated.Reason, cs.State.Terminated.ExitCode, cs.State.Terminated.Message)
	case cs.State.Running != nil:
		if !cs.Ready {
			return "Running (not ready)"
		}
		return "Running (ready)"
	default:
		return ""
	}
}

func printUpgradeFailureDiagnostics(namespace, releaseName string, monitor *upgradePodMonitor, atomic bool, helmErr error, debug bool) {
	printDiagnosticsBanner("UPGRADE FAILED — DIAGNOSTICS")

	if helmErr != nil {
		printDiagnosticsSubSection("Helm Error")
		pterm.Error.Println(helmErr.Error())
	}

	if atomic {
		printDiagnosticsSubSection("Rollback Notice")
		pterm.Warning.Println("Atomic rollback removed failed pods from the cluster.")
		pterm.Warning.Println("Logs and events below were captured while the pod was still running.")
	}

	clientset, err := getKubeClient()
	if err != nil {
		pterm.Error.Printf("Could not connect to cluster for diagnostics: %v\n", err)
		return
	}

	snapshots := monitor.failedSnapshots()
	if len(snapshots) > 0 {
		printDiagnosticsSubSection(fmt.Sprintf("Failed Pods During Rollout (%d)", len(snapshots)))
		for i, snapshot := range snapshots {
			printFailedPodReport(clientset, namespace, snapshot, i+1, len(snapshots))
		}
	}

	printDeploymentRolloutStatus(clientset, namespace, releaseName)

	if len(snapshots) == 0 {
		printDiagnosticsSubSection("Failed Resource Details")
		describeFailedResources(namespace, releaseName)
	}

	printDiagnosticsSubSection("Release Resources")
	printReleaseResources(namespace, releaseName)

	if len(snapshots) == 0 {
		printDiagnosticsSubSection("Current Pod Status")
		pterm.Warning.Println("No failed pod snapshots were captured during the upgrade window.")
		if err := printFinalPodStatus(namespace, releaseName, debug); err != nil {
			pterm.Warning.Printf("Current pod status: %v\n", err)
		}

		livePods, listErr := getPods(namespace, releaseName)
		if listErr == nil {
			for _, pod := range livePods {
				if isPodUnhealthyForUpgrade(&pod) {
					printFailedPodReport(clientset, namespace, failedPodSnapshot{
						pod:   pod.DeepCopy(),
						cache: capturePodDiagnostics(clientset, namespace, &pod),
					}, 1, 1)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
}

func printDeploymentRolloutStatus(clientset *kubernetes.Clientset, namespace, releaseName string) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf(appKubernets, releaseName),
	})
	if err != nil || len(deployments.Items) == 0 {
		return
	}

	printDiagnosticsSubSection("Deployment Rollout Status")
	for _, dep := range deployments.Items {
		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		fmt.Printf("  %s\n", dep.Name)
		fmt.Printf("    Ready     : %d/%d\n", dep.Status.ReadyReplicas, replicas)
		fmt.Printf("    Updated   : %d\n", dep.Status.UpdatedReplicas)
		fmt.Printf("    Available : %d\n", dep.Status.AvailableReplicas)

		for _, cond := range dep.Status.Conditions {
			if cond.Status != corev1.ConditionTrue && cond.Message != "" {
				pterm.Warning.Printf("    Condition : %s (%s) — %s\n", cond.Type, cond.Reason, cond.Message)
			}
		}

		events, evtErr := clientset.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Deployment", dep.Name),
		})
		if evtErr == nil && len(events.Items) > 0 {
			limit := len(events.Items)
			if limit > 5 {
				limit = 5
			}
			fmt.Printf("    Recent events:\n")
			for i := 0; i < limit; i++ {
				evt := events.Items[i]
				prefix := "      ℹ"
				if evt.Type == "Warning" {
					prefix = "      ⚠"
				}
				fmt.Printf("%s  [%s] %s\n", prefix, evt.Reason, evt.Message)
			}
		}
	}
}
