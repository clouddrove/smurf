package helm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// getKubeClient returns a Kubernetes clientset using the kubeconfig file specified in the settings.
func getKubeClient() (*kubernetes.Clientset, error) {
	if kubeClientset != nil {
		return kubeClientset, nil
	}
	config, err := clientcmd.BuildConfigFromFlags("", settings.KubeConfig)
	if err != nil {
		pterm.Error.Println("Failed to build Kubernetes configuration: ", err)
		return nil, fmt.Errorf("failed to build Kubernetes configuration: %v", err)
	}
	kubeClientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		pterm.Error.Println("Failed to create Kubernetes clientset: ", err)
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}
	return kubeClientset, nil
}

// logDetailedError prints a detailed error message based on the error type and provides suggestions for troubleshooting.
// It also prints the resources that failed to be created or updated.
// This function is used to provide more context to the user when an operation fails.
// It prints the error message in red and provides suggestions based on the error type.
func logDetailedError(operation string, err error, namespace, releaseName string) {
	pterm.Error.Printfln("%s FAILED: %v \n", strings.ToUpper(operation), err)

	switch {
	case strings.Contains(err.Error(), "context deadline exceeded"):
		pterm.FgYellow.Printfln("Timeout Suggestions: ")
		pterm.FgYellow.Printfln("- Increase operation timeout using the '--timeout' flag")
		pterm.FgYellow.Printfln("- Check cluster resource availability and networking")
		pterm.FgYellow.Printfln("- Ensure the cluster is not overloaded")
	case strings.Contains(err.Error(), "connection refused"):
		pterm.FgYellow.Printfln("Connection Suggestions: ")
		pterm.FgYellow.Printfln("- Verify cluster connectivity ")
		pterm.FgYellow.Printfln("- Check the kubeconfig file and cluster endpoint ")
		pterm.FgYellow.Printfln("- Ensure the Kubernetes API server is reachable ")
	case strings.Contains(err.Error(), "no matches for kind"),
		strings.Contains(err.Error(), "failed to create"),
		strings.Contains(err.Error(), "YAML parse error"):
		pterm.FgYellow.Printfln("Chart/Configuration Suggestions: \n")
		pterm.FgYellow.Printfln("- Run 'helm lint' on your chart to detect errors. \n")
		pterm.FgYellow.Printfln("- Check if your CRDs or resources exist on the cluster. \n")
		pterm.FgYellow.Printfln("- Validate your values files for incorrect syntax. \n")
	}

	describeFailedResources(namespace, releaseName)
}

// debugLog prints a debug log message to the console.
// This function is used for debugging purposes to print additional information during execution
func debugLog(format string, v ...interface{}) {
	pterm.Debug.Printf(format, v...)
}

// logOperation prints consistent operation logs with timing
func logOperation(debug bool, operation string, args ...interface{}) {
	if debug {
		pterm.Info.Printf("[%s] %s\n", time.Now().Format("15:04:05.000"), fmt.Sprintf(operation, args...))
	}
}

func ensureNamespace(namespace string, create bool) error {
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if apierrors.IsNotFound(err) {
		if create {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}
			_, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create namespace '%s': %v", namespace, err)
			}
			return nil
		}
		return fmt.Errorf("namespace '%s' does not exist and was not created", namespace)
	}

	// Unknown error
	return fmt.Errorf("error checking namespace '%s': %v", namespace, err)
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if vMap, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bvMap, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bvMap, vMap)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// printReleaseInfo prints detailed information about the specified Helm release.
func printReleaseInfo(rel *release.Release, debug bool) {
	logOperation(debug, "Printing release info for %s", rel.Name)

	// Clean header with release name
	pterm.Println()
	pterm.Printf("%s %s\n\n",
		pterm.Bold.Sprint("RELEASE :"),
		pterm.Cyan(rel.Name))

	// Release information in a clean table
	releaseTable := pterm.TableData{
		{"NAME", rel.Name},
		{"CHART", fmt.Sprintf("%s-%s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version)},
		{"NAMESPACE", rel.Namespace},
		{"LAST DEPLOYED", rel.Info.LastDeployed.Format("2006-01-02 15:04:05")},
		{"STATUS", string(rel.Info.Status)},
		{"REVISION", fmt.Sprintf("%d", rel.Version)},
	}

	pterm.DefaultTable.
		WithBoxed(true).
		WithHeaderStyle(pterm.NewStyle(pterm.Bold)).
		WithLeftAlignment().
		WithData(releaseTable).
		Render()

	// Application notes section (completely separate from table)
	if rel.Info.Notes != "" {
		pterm.Println("\n" + pterm.DefaultSection.
			WithLevel(2).
			WithStyle(pterm.NewStyle(pterm.Bold)).
			Sprint("NOTES : "))

		// Print raw notes without table formatting
		fmt.Println(rel.Info.Notes)
	}
	pterm.Println()
}

// convertToMapStringInterface converts an interface{} to a map[string]interface{} recursively.
// This function is used to convert the raw YAML object to a map for easier parsing.
// It handles nested maps and arrays by recursively converting the elements.
func convertToMapStringInterface(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, v := range x {
			m[fmt.Sprintf("%v", k)] = convertToMapStringInterface(v)
		}
		return m
	case []interface{}:
		for i, v := range x {
			x[i] = convertToMapStringInterface(v)
		}
	}
	return i
}

// parseResourcesFromManifest parses the Kubernetes resources from the manifest string.
// It returns a slice of Resource objects containing the kind and name of each resource.
// This function is used to extract the resources created by a Helm release for monitoring.
func parseResourcesFromManifest(manifest string) ([]Resource, error) {
	var resources []Resource
	docs := strings.Split(manifest, "---")
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var rawObj interface{}
		err := yaml.Unmarshal([]byte(doc), &rawObj)
		if err != nil {
			pterm.Error.Printfln("failed to parse manifest: %v", err)
			return nil, fmt.Errorf("failed to parse manifest: %v", err)
		}

		obj := convertToMapStringInterface(rawObj).(map[string]interface{})
		kind, _ := obj["kind"].(string)
		metadata, _ := obj["metadata"].(map[string]interface{})
		if kind == "" || metadata == nil {
			continue
		}

		name, _ := metadata["name"].(string)
		if kind != "" && name != "" {
			resources = append(resources, Resource{Kind: kind, Name: name})
		}
	}
	return resources, nil
}

// monitorResources monitors the resources created by the Helm release until they are all ready.
// It checks the status of the resources in the Kubernetes API and waits until they are all ready.
// The function returns an error if the resources are not ready within the specified timeout.
func monitorResources(rel *release.Release, namespace string, timeout time.Duration) (err error) {
	resources, err := parseResourcesFromManifest(rel.Manifest)
	if err != nil {
		return err
	}

	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	spinner, _ := pterm.DefaultSpinner.Start("Checking resource readiness... \n")
	defer spinner.Stop()

	deadline := time.Now().Add(timeout)
	for {
		allReady, notReadyResources, err := resourcesReady(clientset, namespace, resources)
		if err != nil {
			return err
		}
		if allReady {
			spinner.Success("All resources are ready. \n")
			return nil
		}
		if time.Now().After(deadline) {
			spinner.Fail("Timeout waiting for all resources to become ready \n")
			return errors.New("timeout waiting for all resources to become ready")
		}

		spinner.UpdateText(fmt.Sprintf("Waiting for resources: %s \n", strings.Join(notReadyResources, ", ")))

		time.Sleep(5 * time.Second)
	}
}

// resourcesReady checks if the specified resources are ready in the Kubernetes API.
// It returns a boolean indicating if all resources are ready, a slice of not ready resources, and an error if any.
// The function checks the status of Deployments and Pods to determine if they are ready.
func resourcesReady(clientset *kubernetes.Clientset, namespace string, resources []Resource) (bool, []string, error) {
	var notReadyResources []string

	for _, res := range resources {
		switch res.Kind {
		case "Deployment":
			dep, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), res.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Error.Println(err)
				return false, nil, err
			}
			if dep.Status.ReadyReplicas != *dep.Spec.Replicas {
				notReadyResources = append(notReadyResources, fmt.Sprintf("Deployment/%s (%d/%d)", res.Name, dep.Status.ReadyReplicas, *dep.Spec.Replicas))
			}
		case "Pod":
			pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), res.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Error.Println(err)
				return false, nil, err
			}
			if pod.Status.Phase != corev1.PodRunning {
				notReadyResources = append(notReadyResources, fmt.Sprintf("Pod/%s (Phase: %s)", res.Name, pod.Status.Phase))
			} else {
				ready := false
				for _, cond := range pod.Status.Conditions {
					if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
						ready = true
						break
					}
				}
				if !ready {
					notReadyResources = append(notReadyResources, fmt.Sprintf("Pod/%s (Not Ready)", res.Name))
				}
			}
		}
	}

	if len(notReadyResources) == 0 {
		return true, nil, nil
	}
	pterm.Info.Printfln("All resource ready...")
	return false, notReadyResources, nil
}

// describeFailedResources fetches detailed information about the failed resources in the Helm release.
// It retrieves the pods associated with the release and prints their status and events for troubleshooting.
// This function is used to provide additional context to the user when resources fail to be created or updated.
func describeFailedResources(namespace, releaseName string) {
	pterm.FgCyan.Print("----- TROUBLESHOOTING FAILED RESOURCES ----- \n")
	clientset, err := getKubeClient()
	if err != nil {
		pterm.Error.Printfln("Error getting kube client: %v \n", err)
		return
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		pterm.Error.Printfln("Failed to list pods for troubleshooting: %v \n", err)
		return
	}

	if len(podList.Items) == 0 {
		pterm.Warning.Printfln("No pods found for release '%s', cannot diagnose further.\n", releaseName)
		return
	}

	for _, pod := range podList.Items {
		pterm.FgGreen.Printfln("Pod: %s \n", pod.Name)
		pterm.FgGreen.Printfln("Phase: %s \n", pod.Status.Phase)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				pterm.FgRed.Printfln("Container: %s is waiting with reason: %s, message: %s \n", cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message)
			} else if cs.State.Terminated != nil {
				pterm.FgRed.Printfln("Container: %s is terminated with reason: %s, message: %s \n", cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.Message)
			}
		}

		evts, err := clientset.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
		})
		if err != nil {
			pterm.Warning.Printfln("Error fetching events for pod %s: %v \n", pod.Name, err)
			continue
		}

		if len(evts.Items) == 0 {
			pterm.Warning.Printfln("No events found for pod %s \n", pod.Name)
		} else {
			pterm.FgGreen.Printfln("Events for pod %s: \n", pod.Name)
			for _, evt := range evts.Items {
				pterm.FgGreen.Printfln("  %s: %s \n", evt.Reason, evt.Message)
			}
		}
		pterm.FgCyan.Println("-------------------------------------------------------")
	}
	pterm.FgCyan.Println("-----------------------------------------------")
}

// Helper functions
func isNotFound(err error) bool {
	return err != nil && (errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "not found"))
}

// Helper function to get external IP for services
func getExternalIP(svc *corev1.Service) string {
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		for _, ing := range svc.Status.LoadBalancer.Ingress {
			if ing.IP != "" {
				return ing.IP
			}
			if ing.Hostname != "" {
				return ing.Hostname
			}
		}
		return "<pending>"
	}
	return "<none>"
}

// printResourcesFromRelease prints detailed information about the Kubernetes resources created by the Helm release.
// It fetches detailed information about the resources from the Kubernetes API and prints it to the console.
func printResourcesFromRelease(rel *release.Release) {
	resources, err := parseResourcesFromManifest(rel.Manifest)
	if err != nil {
		pterm.Warning.Printfln("Error parsing manifest: %v", err)
		return
	}

	if len(resources) == 0 {
		pterm.Info.Println("No Kubernetes resources were created by this release.")
		return
	}

	// Simple resource list
	pterm.DefaultSection.Println("CREATED RESOURCES")

	// Group by kind
	resourcesByKind := make(map[string][]string)
	for _, r := range resources {
		resourcesByKind[r.Kind] = append(resourcesByKind[r.Kind], r.Name)
	}

	// Print simple list
	for kind, names := range resourcesByKind {
		pterm.Info.Printf("%s:\n", kind)
		for _, name := range names {
			pterm.Printf("  - %s\n", name)
		}
		pterm.Println()
	}

	// Simple pod status - try multiple ways to find pods
	pterm.DefaultSection.Println("POD STATUS")

	clientset, err := getKubeClient()
	if err != nil {
		pterm.Warning.Printfln("Cannot get pod status: %v", err)
		return
	}

	// Try multiple label selectors to find pods
	var podList *corev1.PodList
	selectors := []string{
		fmt.Sprintf("app.kubernetes.io/instance=%s", rel.Name),
		fmt.Sprintf("release=%s", rel.Name),
		fmt.Sprintf("app=%s", rel.Name),
	}

	for _, selector := range selectors {
		podList, err = clientset.CoreV1().Pods(rel.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err == nil && len(podList.Items) > 0 {
			pterm.Debug.Printfln("Found pods using selector: %s", selector)
			break
		}
	}

	// If no pods found with label selectors, list all pods in namespace and filter by owner
	if podList == nil || len(podList.Items) == 0 {
		podList, err = clientset.CoreV1().Pods(rel.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			pterm.Warning.Printfln("Error listing all pods: %v", err)
			return
		}

		// Filter pods that might belong to this release by checking if their name contains release name
		var filteredPods []corev1.Pod
		for _, pod := range podList.Items {
			status := string(pod.Status.Phase)
			isFailed := false
			failureReason := ""

			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					if cs.State.Waiting.Reason == "ImagePullBackOff" || cs.State.Waiting.Reason == "ErrImagePull" {
						isFailed = true
						failureReason = cs.State.Waiting.Reason
						status = fmt.Sprintf("Failed (%s)", cs.State.Waiting.Reason)
						break
					}
				}
			}

			if isFailed {
				pterm.Error.Printf("• %s: %s - %s\n", pod.Name, status, failureReason)
			} else {
				pterm.Printf("• %s: %s%s", pod.Name, status)
			}
		}

		podList.Items = filteredPods
	}

	if len(podList.Items) == 0 {
		pterm.Info.Println("No pods found for this release.")
		pterm.Info.Println("This could be because:")
		pterm.Info.Println("  - Pods haven't been created yet (use --wait to wait for creation)")
		pterm.Info.Println("  - The release doesn't create pods directly")
		pterm.Info.Println("  - Pods have different labels")
		return
	}

	pterm.Info.Printf("Found %d pod(s):\n", len(podList.Items))
	for _, pod := range podList.Items {
		status := string(pod.Status.Phase)

		// Check if pod is ready
		ready := false
		readyContainers := 0
		totalContainers := len(pod.Spec.Containers)

		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				ready = true
			}
		}

		// Count ready containers
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
		}

		if ready {
			status = "Ready"
		} else if pod.Status.Phase == corev1.PodRunning {
			status = fmt.Sprintf("Running (%d/%d)", readyContainers, totalContainers)
		}

		restartCount := int32(0)
		if len(pod.Status.ContainerStatuses) > 0 {
			restartCount = pod.Status.ContainerStatuses[0].RestartCount
		}

		// Show pod age if available
		age := ""
		if pod.CreationTimestamp.Time != (time.Time{}) {
			age = fmt.Sprintf(" (%s ago)", time.Since(pod.CreationTimestamp.Time).Round(time.Second))
		}

		pterm.Printf("• %s: %s%s", pod.Name, status, age)
		if restartCount > 0 {
			pterm.Printf(" - Restarts: %d", restartCount)
		}
		pterm.Println()

		// Show container status if not all are ready
		if readyContainers < totalContainers {
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					if cs.State.Waiting != nil {
						pterm.Printf("  └─ %s: %s - %s\n", cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message)
					} else if cs.State.Terminated != nil {
						pterm.Printf("  └─ %s: Terminated - %s\n", cs.Name, cs.State.Terminated.Reason)
					}
				}
			}
		}
	}
	pterm.Println()
}
