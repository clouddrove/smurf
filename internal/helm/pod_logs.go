package helm

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pterm/pterm"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const defaultPodLogTailLines int64 = 100

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

// printFailedPodLogs prints pod logs in kubectl-style sections (current + --previous when needed).
func printFailedPodLogs(clientset *kubernetes.Clientset, namespace string, pod corev1.Pod) {
	targets := containersForLogFetch(&pod)
	if len(targets) == 0 {
		pterm.Warning.Printf("No containers to fetch logs for pod %s\n", pod.Name)
		return
	}

	multiContainer := len(pod.Spec.Containers) > 1

	for _, target := range targets {
		containerLabel := target.name
		if target.init {
			containerLabel = target.name + " (init)"
		}

		// Current instance logs (kubectl logs podname)
		containerArg := ""
		if multiContainer || target.init {
			containerArg = target.name
		}
		currentCmd := kubectlLogsCommand(namespace, pod.Name, containerArg, false, defaultPodLogTailLines)

		title := fmt.Sprintf("Logs — %s", containerLabel)
		printLogSectionHeader(title, currentCmd)

		logs, err := fetchPodContainerLogs(clientset, namespace, pod.Name, podLogFetchOpts{
			container: target.name,
			previous:  false,
			tailLines: defaultPodLogTailLines,
		})
		printLogSectionBody(logs, err)

		// Previous instance logs (kubectl logs podname --previous) for crash loops
		if target.previous {
			prevCmd := kubectlLogsCommand(namespace, pod.Name, target.name, true, defaultPodLogTailLines)
			printLogSectionHeader(fmt.Sprintf("Previous logs — %s", containerLabel), prevCmd)

			prevLogs, prevErr := fetchPodContainerLogs(clientset, namespace, pod.Name, podLogFetchOpts{
				container: target.name,
				previous:  true,
				tailLines: defaultPodLogTailLines,
			})
			printLogSectionBody(prevLogs, prevErr)
		}
	}
}

// getPodLogs kept for callers that only need raw log text.
func getPodLogs(clientset *kubernetes.Clientset, namespace, podName, containerName string, tailLines int64) (string, error) {
	return fetchPodContainerLogs(clientset, namespace, podName, podLogFetchOpts{
		container: containerName,
		tailLines: tailLines,
	})
}
