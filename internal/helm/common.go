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
		pterm.Info.Println("Successfuly Kubernetes client set")
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
	fmt.Printf(format, v...)
	fmt.Println()
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
		pterm.Success.Sprintf("Namespace '%s' already exists.\n", namespace)
		return nil
	}
	if apierrors.IsNotFound(err) {
		if create {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace},
			}
			_, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			if err != nil {
				pterm.Error.Printf("Failed to create namespace '%s': %v\n", namespace, err)
				return fmt.Errorf("failed to create namespace '%s': %v", namespace, err)
			}
			pterm.Success.Printf("Namespace '%s' created successfully.\n", namespace)
			return nil
		}
		pterm.Error.Printf("namespace '%v' does not exist and was not created\n", namespace)
		return fmt.Errorf("namespace '%s' does not exist and was not created", namespace)
	}

	// Unknown error
	pterm.Error.Printf("namespace '%v' does not exist and was not created\n", namespace)
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

	pterm.FgCyan.Println("\n----- RELEASE INFO -----")

	pterm.FgGreen.Printf("NAME: %s\n", rel.Name)
	pterm.FgGreen.Printf("CHART: %s-%s\n", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version)
	pterm.FgGreen.Printf("NAMESPACE: %s\n", rel.Namespace)
	pterm.FgGreen.Printf("LAST DEPLOYED: %s\n\n", rel.Info.LastDeployed)
	pterm.FgGreen.Printf("STATUS: %s\n", rel.Info.Status)
	pterm.FgGreen.Printf("REVISION: %d\n", rel.Version)

	if rel.Info.Notes != "" {
		logOperation(debug, "Release notes available")
		pterm.FgGreen.Println("\nNOTES:")
		fmt.Println(rel.Info.Notes)
	}

	pterm.FgCyan.Println("-----------------------")
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

// printResourcesFromRelease prints detailed information about the Kubernetes resources created by the Helm release.
// It fetches detailed information about the resources from the Kubernetes API and prints it to the console.
func printResourcesFromRelease(rel *release.Release) {
	resources, err := parseResourcesFromManifest(rel.Manifest)
	if err != nil {
		pterm.Error.Printfln("Error parsing manifest: %v \n", err)
		return
	}

	if len(resources) == 0 {
		pterm.FgGreen.Println("No Kubernetes resources were created by this release.")
		return
	}

	pterm.FgCyan.Print("----- RESOURCES ----- \n")

	clientset, getClientErr := getKubeClient()
	if getClientErr != nil {
		pterm.Error.Printfln("Error getting kube client for detailed resource info: %v \n", getClientErr)
		for _, r := range resources {
			pterm.FgGreen.Printfln("%s: %s \n", r.Kind, r.Name)
		}
		pterm.FgCyan.Print("-------------------------------- \n")
		return
	}

	for _, r := range resources {
		switch r.Kind {
		case "Deployment":
			dep, err := clientset.AppsV1().Deployments(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("Deployment: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Desired Replicas: %d \n", *dep.Spec.Replicas)
			pterm.FgGreen.Printfln("- Ready Replicas:   %d \n", dep.Status.ReadyReplicas)

		case "ReplicaSet":
			rs, err := clientset.AppsV1().ReplicaSets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("ReplicaSet: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Desired Replicas: %d \n", *rs.Spec.Replicas)
			pterm.FgGreen.Printfln("- Current Replicas: %d \n", rs.Status.Replicas)
			pterm.FgGreen.Printfln("- Ready Replicas:   %d \n", rs.Status.ReadyReplicas)

		case "StatefulSet":
			ss, err := clientset.AppsV1().StatefulSets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("StatefulSet: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Desired Replicas: %d \n", *ss.Spec.Replicas)
			pterm.FgGreen.Printfln("- Current Replicas: %d \n", ss.Status.CurrentReplicas)
			pterm.FgGreen.Printfln("- Ready Replicas:   %d \n", ss.Status.ReadyReplicas)

		case "DaemonSet":
			ds, err := clientset.AppsV1().DaemonSets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("DaemonSet: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Desired Number Scheduled: %d \n", ds.Status.DesiredNumberScheduled)
			pterm.FgGreen.Printfln("- Number Scheduled:         %d \n", ds.Status.CurrentNumberScheduled)
			pterm.FgGreen.Printfln("- Number Ready:             %d \n", ds.Status.NumberReady)

		case "Pod":
			pod, err := clientset.CoreV1().Pods(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("Pod: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Phase: %s \n", pod.Status.Phase)
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			pterm.FgGreen.Printfln("- Ready: %v \n", ready)
			pterm.FgGreen.Printfln("- IP: %s \n", pod.Status.PodIP)
			for _, cs := range pod.Status.ContainerStatuses {
				pterm.FgGreen.Printfln("  Container: %s \n", cs.Name)
				if cs.State.Waiting != nil {
					pterm.FgRed.Printfln("    State: Waiting \n")
					pterm.FgRed.Printfln("    Reason: %s \n", cs.State.Waiting.Reason)
					pterm.FgRed.Printfln("    Message: %s \n", cs.State.Waiting.Message)
				}
				if cs.State.Terminated != nil {
					pterm.FgRed.Printfln("    State: Terminated \n")
					pterm.FgRed.Printfln("    Reason: %s \n", cs.State.Terminated.Reason)
					pterm.FgRed.Printfln("    Message: %s \n", cs.State.Terminated.Message)
				}
				if cs.State.Running != nil {
					pterm.FgGreen.Printfln("    State: Running \n")
					pterm.FgGreen.Printfln("    Started at: %s \n", cs.State.Running.StartedAt)
				}
				pterm.FgGreen.Printfln("    Ready: %v \n", cs.Ready)
				pterm.FgGreen.Printfln("    Restart Count: %d \n", cs.RestartCount)
			}

		case "Service":
			svc, err := clientset.CoreV1().Services(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("Service: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Type: %s \n", svc.Spec.Type)
			pterm.FgGreen.Printfln("- ClusterIP: %s \n", svc.Spec.ClusterIP)
			if len(svc.Spec.Ports) > 0 {
				for _, p := range svc.Spec.Ports {
					pterm.FgGreen.Printfln("- Port: %d (Protocol: %s, TargetPort: %v) \n", p.Port, p.Protocol, p.TargetPort)
				}
			}

		case "ServiceAccount":
			sa, err := clientset.CoreV1().ServiceAccounts(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("ServiceAccount: %s \n", r.Name)
			pterm.FgGreen.Printfln("- CreationTimestamp: %s \n", sa.CreationTimestamp.String())

		case "ConfigMap":
			cm, err := clientset.CoreV1().ConfigMaps(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("ConfigMap: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Data keys: %d \n", len(cm.Data))

		case "Secret":
			secret, err := clientset.CoreV1().Secrets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("Secret: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Type: %s \n", secret.Type)
			pterm.FgGreen.Printfln("- Data keys: %d \n", len(secret.Data))

		case "Namespace":
			ns, err := clientset.CoreV1().Namespaces().Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				pterm.Warning.Printfln("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			pterm.FgGreen.Printfln("Namespace: %s \n", r.Name)
			pterm.FgGreen.Printfln("- Status: %s \n", ns.Status.Phase)

		default:
			pterm.FgGreen.Printfln("%s: %s \n", r.Kind, r.Name)
		}
	}

	pterm.FgCyan.Print("----- PODS ASSOCIATED WITH THE RELEASE ----- \n")
	podList, err := clientset.CoreV1().Pods(rel.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", rel.Name),
	})
	if err != nil {
		pterm.Error.Printfln("Error listing pods for release '%s': %v \n", rel.Name, err)
	} else if len(podList.Items) == 0 {
		pterm.FgYellow.Printfln("No pods found for release '%s' \n", rel.Name)
	} else {
		for _, pod := range podList.Items {
			pterm.FgGreen.Printfln("Pod: %s \n", pod.Name)
			pterm.FgGreen.Printfln("- Phase: %s \n", pod.Status.Phase)
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			pterm.FgGreen.Printfln("- Ready: %v \n", ready)
			pterm.FgGreen.Printfln("- IP: %s \n", pod.Status.PodIP)
			for _, cs := range pod.Status.ContainerStatuses {
				pterm.FgGreen.Printfln("  Container: %s \n", cs.Name)
				if cs.State.Waiting != nil {
					pterm.FgRed.Printfln("    State: Waiting \n")
					pterm.FgRed.Printfln("    Reason: %s \n", cs.State.Waiting.Reason)
					pterm.FgRed.Printfln("    Message: %s \n", cs.State.Waiting.Message)
				}
				if cs.State.Terminated != nil {
					pterm.FgRed.Printfln("    State: Terminated \n")
					pterm.FgRed.Printfln("    Reason: %s \n", cs.State.Terminated.Reason)
					pterm.FgRed.Printfln("    Message: %s \n", cs.State.Terminated.Message)
				}
				if cs.State.Running != nil {
					pterm.FgGreen.Printfln("    State: Running \n")
					pterm.FgGreen.Printfln("    Started at: %s \n", cs.State.Running.StartedAt)
				}
				pterm.FgGreen.Printfln("    Ready: %v \n", cs.Ready)
				pterm.FgGreen.Printfln("    Restart Count: %d \n", cs.RestartCount)
			}
			pterm.FgGreen.Printfln("- Node Name: %s \n", pod.Spec.NodeName)
			pterm.FgGreen.Printfln("- Host IP: %s \n", pod.Status.HostIP)
			pterm.FgGreen.Printfln("- Pod IP: %s \n", pod.Status.PodIP)
			pterm.FgGreen.Printfln("- Start Time: %s \n", pod.Status.StartTime.String())

			evts, err := clientset.CoreV1().Events(rel.Namespace).List(context.Background(), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
			})
			if err != nil {
				pterm.FgYellow.Printfln("  Error fetching events for pod %s: %v \n", pod.Name, err)
				continue
			}

			if len(evts.Items) == 0 {
				pterm.FgYellow.Printfln("  No events found for pod %s \n", pod.Name)
			} else {
				pterm.FgGreen.Printfln("  Events for pod %s: \n", pod.Name)
				for _, evt := range evts.Items {
					pterm.FgGreen.Printfln("    %s: %s \n", evt.Reason, evt.Message)
				}
			}
			pterm.FgCyan.Print("------------------------------------------------------- \n")
		}
	}
	pterm.FgCyan.Print("----------------------------------------------- \n")
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
