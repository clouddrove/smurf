package helm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const defaultPodLogTailLines int64 = 100

type cachedPodDiagnostics struct {
	current    map[string]string // container -> current logs
	previous   map[string]string // container -> previous logs
	events     []corev1.Event
	capturedAt time.Time
	podExists  bool
}

func newCachedPodDiagnostics() *cachedPodDiagnostics {
	return &cachedPodDiagnostics{
		current:  make(map[string]string),
		previous: make(map[string]string),
	}
}

type podLogFetchOpts struct {
	container string
	previous  bool
	tailLines int64
}

// fetchPodContainerLogs retrieves logs for a pod container (kubectl logs equivalent).
func fetchPodContainerLogs(
	clientset *kubernetes.Clientset,
	namespace, podName string,
	opts podLogFetchOpts,
) (string, error) {
	tail := opts.tailLines
	if tail <= 0 {
		tail = defaultPodLogTailLines
	}

	podLogOpts := corev1.PodLogOptions{
		Container: opts.container,
		TailLines: &tail,
		Previous:  opts.previous,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)
	stream, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer stream.Close()

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, stream); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func kubectlLogsCommand(namespace, podName, container string, previous bool, tail int64) string {
	parts := []string{"kubectl", "logs", podName, "-n", namespace}
	if container != "" {
		parts = append(parts, "-c", container)
	}
	if previous {
		parts = append(parts, "--previous")
	}
	if tail > 0 {
		parts = append(parts, fmt.Sprintf("--tail=%d", tail))
	}
	return strings.Join(parts, " ")
}

func isContainerStatusUnhealthy(cs corev1.ContainerStatus) bool {
	if cs.State.Waiting != nil {
		switch cs.State.Waiting.Reason {
		case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull",
			"CreateContainerConfigError", "InvalidImageName", "CreateContainerError",
			"RunContainerError", "StartError":
			return true
		}
	}
	if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
		return true
	}
	if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.ExitCode != 0 {
		return true
	}
	return false
}

func containerNeedsPreviousLogs(cs corev1.ContainerStatus) bool {
	if cs.LastTerminationState.Terminated != nil {
		return true
	}
	if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
		return true
	}
	if cs.RestartCount > 0 {
		return true
	}
	return false
}

type containerLogTarget struct {
	name     string
	init     bool
	previous bool
}

func containersForLogFetch(pod *corev1.Pod) []containerLogTarget {
	seen := make(map[string]bool)
	var targets []containerLogTarget

	add := func(name string, init bool, cs corev1.ContainerStatus) {
		key := name
		if init {
			key = "init:" + name
		}
		if seen[key] {
			return
		}
		seen[key] = true
		targets = append(targets, containerLogTarget{
			name:     name,
			init:     init,
			previous: containerNeedsPreviousLogs(cs),
		})
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		if isContainerStatusUnhealthy(cs) || pod.Status.Phase == corev1.PodFailed {
			add(cs.Name, true, cs)
		}
	}

	unhealthyFound := false
	for _, cs := range pod.Status.ContainerStatuses {
		if isContainerStatusUnhealthy(cs) {
			unhealthyFound = true
			add(cs.Name, false, cs)
		}
	}

	if !unhealthyFound {
		if len(pod.Spec.Containers) == 1 {
			name := pod.Spec.Containers[0].Name
			var cs corev1.ContainerStatus
			if len(pod.Status.ContainerStatuses) > 0 {
				cs = pod.Status.ContainerStatuses[0]
			}
			add(name, false, cs)
		} else {
			for i, c := range pod.Spec.Containers {
				var cs corev1.ContainerStatus
				if i < len(pod.Status.ContainerStatuses) {
					cs = pod.Status.ContainerStatuses[i]
				}
				add(c.Name, false, cs)
			}
		}
	}

	return targets
}

func printLogSectionHeader(title, kubectlCmd string) {
	fmt.Println()
	pterm.DefaultSection.WithLevel(2).Println(title)
	fmt.Printf("  %s%s%s\n", pterm.Gray("$ "), pterm.Cyan(kubectlCmd), pterm.Gray(""))
	fmt.Println(strings.Repeat("-", 80))
}

func printLogSectionBody(logs string, fetchErr error) {
	if fetchErr != nil {
		pterm.Warning.Printf("  (logs unavailable: %v)\n", fetchErr)
		fmt.Println(strings.Repeat("-", 80))
		return
	}

	trimmed := strings.TrimRight(logs, "\n")
	if trimmed == "" {
		pterm.Warning.Println("  (no log output)")
	} else {
		fmt.Println(trimmed)
	}
	fmt.Println(strings.Repeat("-", 80))
}

func containerNeverStarted(cs corev1.ContainerStatus) bool {
	if cs.State.Waiting == nil {
		return false
	}
	switch cs.State.Waiting.Reason {
	case "ImagePullBackOff", "ErrImagePull", "InvalidImageName",
		"CreateContainerConfigError", "CreateContainerError", "PodInitializing":
		return true
	}
	return false
}

func capturePodDiagnostics(clientset *kubernetes.Clientset, namespace string, pod *corev1.Pod) *cachedPodDiagnostics {
	cache := newCachedPodDiagnostics()
	cache.capturedAt = time.Now()
	cache.podExists = true

	for _, target := range containersForLogFetch(pod) {
		if logs, err := fetchPodContainerLogs(clientset, namespace, pod.Name, podLogFetchOpts{
			container: target.name,
			previous:  false,
			tailLines: defaultPodLogTailLines,
		}); err == nil && strings.TrimSpace(logs) != "" {
			cache.current[target.name] = logs
		}

		if target.previous {
			if logs, err := fetchPodContainerLogs(clientset, namespace, pod.Name, podLogFetchOpts{
				container: target.name,
				previous:  true,
				tailLines: defaultPodLogTailLines,
			}); err == nil && strings.TrimSpace(logs) != "" {
				cache.previous[target.name] = logs
			}
		}
	}

	if events, err := getPodEvents(clientset, namespace, pod.Name); err == nil {
		cache.events = events
	}

	return cache
}

func mergePodDiagnostics(existing, fresh *cachedPodDiagnostics) *cachedPodDiagnostics {
	if existing == nil {
		return fresh
	}
	if fresh == nil {
		return existing
	}

	for k, v := range fresh.current {
		if v != "" {
			existing.current[k] = v
		}
	}
	for k, v := range fresh.previous {
		if v != "" {
			existing.previous[k] = v
		}
	}
	if len(fresh.events) > 0 {
		existing.events = fresh.events
	}
	if !fresh.capturedAt.IsZero() {
		existing.capturedAt = fresh.capturedAt
	}
	existing.podExists = fresh.podExists
	return existing
}

func printPodEventsSection(events []corev1.Event, title string) {
	if len(events) == 0 {
		return
	}
	fmt.Println()
	pterm.DefaultSection.WithLevel(2).Println(title)
	fmt.Println("  Type    Reason              Message")
	fmt.Println("  ----    ------              -------")
	for _, evt := range events {
		icon := "ℹ"
		if evt.Type == "Warning" {
			icon = "⚠"
		}
		fmt.Printf("  %s %-7s %-19s %s\n", icon, evt.Type, evt.Reason, evt.Message)
	}
	fmt.Println(strings.Repeat("-", 80))
}

func printContainerNeverStartedReason(pod corev1.Pod, containerName string) {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name != containerName {
			continue
		}
		if cs.State.Waiting != nil {
			pterm.Warning.Printf("  Container never started — no runtime logs available.\n")
			pterm.Error.Printf("  Reason  : %s\n", cs.State.Waiting.Reason)
			if cs.State.Waiting.Message != "" {
				pterm.Error.Printf("  Message : %s\n", cs.State.Waiting.Message)
			}
			return
		}
	}
}

// printFailedPodLogs prints pod logs fetched live from the cluster.
func printFailedPodLogs(clientset *kubernetes.Clientset, namespace string, pod corev1.Pod) {
	printFailedPodLogsWithCache(clientset, namespace, pod, nil)
}

// printFailedPodLogsWithCache prints logs from cache (captured during upgrade) with live fallback.
func printFailedPodLogsWithCache(clientset *kubernetes.Clientset, namespace string, pod corev1.Pod, cache *cachedPodDiagnostics) {
	targets := containersForLogFetch(&pod)
	if len(targets) == 0 {
		pterm.Warning.Printf("No containers to fetch logs for pod %s\n", pod.Name)
		return
	}

	multiContainer := len(pod.Spec.Containers) > 1
	podRemoved := cache != nil && !cache.podExists

	for _, target := range targets {
		containerLabel := target.name
		if target.init {
			containerLabel = target.name + " (init)"
		}

		containerArg := ""
		if multiContainer || target.init {
			containerArg = target.name
		}
		currentCmd := kubectlLogsCommand(namespace, pod.Name, containerArg, false, defaultPodLogTailLines)

		title := fmt.Sprintf("Logs — %s", containerLabel)
		if podRemoved {
			title += " (captured before pod removal)"
		}
		printLogSectionHeader(title, currentCmd)

		var logs string
		var fetchErr error

		if cache != nil {
			if cached, ok := cache.current[target.name]; ok && strings.TrimSpace(cached) != "" {
				logs = cached
			}
		}
		if logs == "" && !podRemoved {
			logs, fetchErr = fetchPodContainerLogs(clientset, namespace, pod.Name, podLogFetchOpts{
				container: target.name,
				previous:  false,
				tailLines: defaultPodLogTailLines,
			})
		} else if logs == "" && podRemoved {
			fetchErr = fmt.Errorf("pod %q was removed during rollback (no cached logs)", pod.Name)
		}

		if logs != "" {
			printLogSectionBody(logs, nil)
		} else if containerNeverStarted(findContainerStatus(pod, target.name)) {
			printContainerNeverStartedReason(pod, target.name)
			if cache != nil && len(cache.events) > 0 {
				printPodEventsSection(cache.events, "Pod Events (captured during failure)")
			} else if events, err := getPodEvents(clientset, namespace, pod.Name); err == nil {
				printPodEventsSection(events, "Pod Events")
			}
			fmt.Println(strings.Repeat("-", 80))
		} else {
			printLogSectionBody("", fetchErr)
			if cache != nil && len(cache.events) > 0 {
				printPodEventsSection(cache.events, "Pod Events (captured during failure)")
			}
		}

		if target.previous {
			prevCmd := kubectlLogsCommand(namespace, pod.Name, target.name, true, defaultPodLogTailLines)
			prevTitle := fmt.Sprintf("Previous logs — %s", containerLabel)
			if podRemoved {
				prevTitle += " (captured before pod removal)"
			}
			printLogSectionHeader(prevTitle, prevCmd)

			var prevLogs string
			var prevErr error
			if cache != nil {
				if cached, ok := cache.previous[target.name]; ok && strings.TrimSpace(cached) != "" {
					prevLogs = cached
				}
			}
			if prevLogs == "" && !podRemoved {
				prevLogs, prevErr = fetchPodContainerLogs(clientset, namespace, pod.Name, podLogFetchOpts{
					container: target.name,
					previous:  true,
					tailLines: defaultPodLogTailLines,
				})
			}
			printLogSectionBody(prevLogs, prevErr)
		}
	}
}

func findContainerStatus(pod corev1.Pod, name string) corev1.ContainerStatus {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == name {
			return cs
		}
	}
	return corev1.ContainerStatus{Name: name}
}

// getPodLogs kept for callers that only need raw log text.
func getPodLogs(clientset *kubernetes.Clientset, namespace, podName, containerName string, tailLines int64) (string, error) {
	return fetchPodContainerLogs(clientset, namespace, podName, podLogFetchOpts{
		container: containerName,
		tailLines: tailLines,
	})
}
