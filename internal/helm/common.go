package helm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
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
		color.Red("Failed to build Kubernetes configuration: %v \n", err)
		return nil, err
	}

	kubeClientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		color.Red("Failed to create Kubernetes clientset: %v \n", err)
		return nil, err
	}
	return kubeClientset, nil
}

// logDetailedError prints a detailed error message based on the error type and provides suggestions for troubleshooting.
// It also prints the resources that failed to be created or updated.
// This function is used to provide more context to the user when an operation fails.
// It prints the error message in red and provides suggestions based on the error type.
func logDetailedError(operation string, err error, namespace, releaseName string) {
	color.Red("%s FAILED: %v \n", strings.ToUpper(operation), err)

	switch {
	case strings.Contains(err.Error(), "context deadline exceeded"):
		color.Yellow("Timeout Suggestions: \n")
		color.Yellow("- Increase operation timeout using the '--timeout' flag \n")
		color.Yellow("- Check cluster resource availability and networking \n")
		color.Yellow("- Ensure the cluster is not overloaded \n")
	case strings.Contains(err.Error(), "connection refused"):
		color.Yellow("Connection Suggestions: \n")
		color.Yellow("- Verify cluster connectivity \n")
		color.Yellow("- Check the kubeconfig file and cluster endpoint \n")
		color.Yellow("- Ensure the Kubernetes API server is reachable \n")
	case strings.Contains(err.Error(), "no matches for kind"),
		strings.Contains(err.Error(), "failed to create"),
		strings.Contains(err.Error(), "YAML parse error"):
		color.Yellow("Chart/Configuration Suggestions: \n")
		color.Yellow("- Run 'helm lint' on your chart to detect errors. \n")
		color.Yellow("- Check if your CRDs or resources exist on the cluster. \n")
		color.Yellow("- Validate your values files for incorrect syntax. \n")
	}

	describeFailedResources(namespace, releaseName)
}

// debugLog prints a debug log message to the console.
// This function is used for debugging purposes to print additional information during execution
func debugLog(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	fmt.Println()
}

// ensureNamespace checks if the specified namespace exists and creates it if it does not.
// If the 'create' flag is set to false, it returns an error if the namespace does not exist.
func ensureNamespace(namespace string, create bool) error {
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	_, err = clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err == nil {
		color.Green("Namespace '%s' already exists.  \n", namespace)
		return nil
	}

	if create {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		_, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		if err != nil {
			color.Red("Failed to create namespace '%s': %v \n", namespace, err)
			return fmt.Errorf("failed to create namespace '%s': %v \n", namespace, err)
		}
		color.Green("Namespace '%s' created successfully. \n", namespace)
	} else {
		return fmt.Errorf("namespace '%s' does not exist and was not created \n", namespace)
	}

	return nil
}

// loadAndMergeValues loads values from the specified files and merges them into a single map.
// It returns the merged values map or an error if the values cannot be loaded.
func loadAndMergeValues(valuesFiles []string) (map[string]interface{}, error) {
	vals := make(map[string]interface{})
	for _, f := range valuesFiles {
		color.Green("Loading values from file: %s \n", f)
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			color.Red("Error reading values file %s: %v \n", f, err)
			return nil, err
		}
		for key, value := range additionalVals {
			vals[key] = value
		}
	}
	return vals, nil
}

// loadAndMergeValuesWithSets loads values from the specified files and merges them with the set values.
// It returns the merged values map or an error if the values cannot be loaded or parsed.
// The set values are applied after loading the values from the files.
func loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteralValues []string) (map[string]interface{}, error) {
	// Debugging: Print parsed arguments
	color.Yellow("Values files provided: %v", valuesFiles)
	color.Yellow("--set values provided: %v", setValues)
	color.Yellow("--set-literal values provided: %v", setLiteralValues)

	// Ensure that valuesFiles are correctly handled
	if len(valuesFiles) == 0 {
		color.Red("")
	}

	// Load values from files
	vals, err := loadAndMergeValues(valuesFiles)
	if err != nil {
		return nil, err
	}

	// Process --set values (structured data)
	for _, set := range setValues {
		color.Green("Parsing --set value: %s", set)
		if err := strvals.ParseInto(set, vals); err != nil {
			color.Red("Error parsing --set value '%s': %v", set, err)
			return nil, err
		}
	}

	// Ensure --set-literal does not contain values files
	cleanedSetLiteralValues := []string{}
	for _, setLiteral := range setLiteralValues {
		if strings.HasSuffix(setLiteral, ".yaml") || strings.HasSuffix(setLiteral, ".yml") {
			color.Red("")
			continue
		}
		cleanedSetLiteralValues = append(cleanedSetLiteralValues, setLiteral)
	}

	// Process cleaned --set-literal values (always as string)
	for _, setLiteral := range cleanedSetLiteralValues {
		color.Green("Parsing --set-literal value: %s", setLiteral)
		if err := strvals.ParseIntoString(setLiteral, vals); err != nil {
			color.Red("Error parsing --set-literal value '%s': %v", setLiteral, err)
			return nil, err
		}
	}

	return vals, nil
}

// printReleaseInfo prints detailed information about the specified Helm release.
func printReleaseInfo(rel *release.Release) {
	color.Cyan("----- RELEASE INFO ----- \n")
	color.Green("NAME: %s \n", rel.Name)
	color.Green("CHART: %s-%s \n", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version)
	color.Green("NAMESPACE: %s \n", rel.Namespace)
	color.Green("LAST DEPLOYED: %s \n", rel.Info.LastDeployed)
	color.Green("STATUS: %s \n", rel.Info.Status)
	color.Green("REVISION: %d \n", rel.Version)
	if rel.Info.Notes != "" {
		color.Green("NOTES:\n%s \n", rel.Info.Notes)
	}
	color.Cyan("------------------------ \n")
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
		color.Red("Error parsing manifest: %v \n", err)
		return
	}

	if len(resources) == 0 {
		color.Green("No Kubernetes resources were created by this release.\n")
		return
	}

	color.Cyan("----- RESOURCES ----- \n")

	clientset, getClientErr := getKubeClient()
	if getClientErr != nil {
		color.Red("Error getting kube client for detailed resource info: %v \n", getClientErr)
		for _, r := range resources {
			color.Green("%s: %s \n", r.Kind, r.Name)
		}
		color.Cyan("-------------------------------- \n")
		return
	}

	for _, r := range resources {
		switch r.Kind {
		case "Deployment":
			dep, err := clientset.AppsV1().Deployments(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("Deployment: %s \n", r.Name)
			color.Green("- Desired Replicas: %d \n", *dep.Spec.Replicas)
			color.Green("- Ready Replicas:   %d \n", dep.Status.ReadyReplicas)

		case "ReplicaSet":
			rs, err := clientset.AppsV1().ReplicaSets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("ReplicaSet: %s \n", r.Name)
			color.Green("- Desired Replicas: %d \n", *rs.Spec.Replicas)
			color.Green("- Current Replicas: %d \n", rs.Status.Replicas)
			color.Green("- Ready Replicas:   %d \n", rs.Status.ReadyReplicas)

		case "StatefulSet":
			ss, err := clientset.AppsV1().StatefulSets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("StatefulSet: %s \n", r.Name)
			color.Green("- Desired Replicas: %d \n", *ss.Spec.Replicas)
			color.Green("- Current Replicas: %d \n", ss.Status.CurrentReplicas)
			color.Green("- Ready Replicas:   %d \n", ss.Status.ReadyReplicas)

		case "DaemonSet":
			ds, err := clientset.AppsV1().DaemonSets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("DaemonSet: %s \n", r.Name)
			color.Green("- Desired Number Scheduled: %d \n", ds.Status.DesiredNumberScheduled)
			color.Green("- Number Scheduled:         %d \n", ds.Status.CurrentNumberScheduled)
			color.Green("- Number Ready:             %d \n", ds.Status.NumberReady)

		case "Pod":
			pod, err := clientset.CoreV1().Pods(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("Pod: %s \n", r.Name)
			color.Green("- Phase: %s \n", pod.Status.Phase)
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			color.Green("- Ready: %v \n", ready)
			color.Green("- IP: %s \n", pod.Status.PodIP)
			for _, cs := range pod.Status.ContainerStatuses {
				color.Green("  Container: %s \n", cs.Name)
				if cs.State.Waiting != nil {
					color.Red("    State: Waiting \n")
					color.Red("    Reason: %s \n", cs.State.Waiting.Reason)
					color.Red("    Message: %s \n", cs.State.Waiting.Message)
				}
				if cs.State.Terminated != nil {
					color.Red("    State: Terminated \n")
					color.Red("    Reason: %s \n", cs.State.Terminated.Reason)
					color.Red("    Message: %s \n", cs.State.Terminated.Message)
				}
				if cs.State.Running != nil {
					color.Green("    State: Running \n")
					color.Green("    Started at: %s \n", cs.State.Running.StartedAt)
				}
				color.Green("    Ready: %v \n", cs.Ready)
				color.Green("    Restart Count: %d \n", cs.RestartCount)
			}

		case "Service":
			svc, err := clientset.CoreV1().Services(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("Service: %s \n", r.Name)
			color.Green("- Type: %s \n", svc.Spec.Type)
			color.Green("- ClusterIP: %s \n", svc.Spec.ClusterIP)
			if len(svc.Spec.Ports) > 0 {
				for _, p := range svc.Spec.Ports {
					color.Green("- Port: %d (Protocol: %s, TargetPort: %v) \n", p.Port, p.Protocol, p.TargetPort)
				}
			}

		case "ServiceAccount":
			sa, err := clientset.CoreV1().ServiceAccounts(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("ServiceAccount: %s \n", r.Name)
			color.Green("- CreationTimestamp: %s \n", sa.CreationTimestamp.String())

		case "ConfigMap":
			cm, err := clientset.CoreV1().ConfigMaps(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("ConfigMap: %s \n", r.Name)
			color.Green("- Data keys: %d \n", len(cm.Data))

		case "Secret":
			secret, err := clientset.CoreV1().Secrets(rel.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("Secret: %s \n", r.Name)
			color.Green("- Type: %s \n", secret.Type)
			color.Green("- Data keys: %d \n", len(secret.Data))

		case "Namespace":
			ns, err := clientset.CoreV1().Namespaces().Get(context.Background(), r.Name, metav1.GetOptions{})
			if err != nil {
				color.Red("%s: %s (Failed to get details: %v) \n", r.Kind, r.Name, err)
				continue
			}
			color.Green("Namespace: %s \n", r.Name)
			color.Green("- Status: %s \n", ns.Status.Phase)

		default:
			color.Green("%s: %s \n", r.Kind, r.Name)
		}
	}

	color.Cyan("----- PODS ASSOCIATED WITH THE RELEASE ----- \n")
	podList, err := clientset.CoreV1().Pods(rel.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", rel.Name),
	})
	if err != nil {
		color.Red("Error listing pods for release '%s': %v \n", rel.Name, err)
	} else if len(podList.Items) == 0 {
		color.Yellow("No pods found for release '%s' \n", rel.Name)
	} else {
		for _, pod := range podList.Items {
			color.Green("Pod: %s \n", pod.Name)
			color.Green("- Phase: %s \n", pod.Status.Phase)
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
					ready = true
					break
				}
			}
			color.Green("- Ready: %v \n", ready)
			color.Green("- IP: %s \n", pod.Status.PodIP)
			for _, cs := range pod.Status.ContainerStatuses {
				color.Green("  Container: %s \n", cs.Name)
				if cs.State.Waiting != nil {
					color.Red("    State: Waiting \n")
					color.Red("    Reason: %s \n", cs.State.Waiting.Reason)
					color.Red("    Message: %s \n", cs.State.Waiting.Message)
				}
				if cs.State.Terminated != nil {
					color.Red("    State: Terminated \n")
					color.Red("    Reason: %s \n", cs.State.Terminated.Reason)
					color.Red("    Message: %s \n", cs.State.Terminated.Message)
				}
				if cs.State.Running != nil {
					color.Green("    State: Running \n")
					color.Green("    Started at: %s \n", cs.State.Running.StartedAt)
				}
				color.Green("    Ready: %v \n", cs.Ready)
				color.Green("    Restart Count: %d \n", cs.RestartCount)
			}
			color.Green("- Node Name: %s \n", pod.Spec.NodeName)
			color.Green("- Host IP: %s \n", pod.Status.HostIP)
			color.Green("- Pod IP: %s \n", pod.Status.PodIP)
			color.Green("- Start Time: %s \n", pod.Status.StartTime.String())

			evts, err := clientset.CoreV1().Events(rel.Namespace).List(context.Background(), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
			})
			if err != nil {
				color.Yellow("  Error fetching events for pod %s: %v \n", pod.Name, err)
				continue
			}

			if len(evts.Items) == 0 {
				color.Yellow("  No events found for pod %s \n", pod.Name)
			} else {
				color.Green("  Events for pod %s: \n", pod.Name)
				for _, evt := range evts.Items {
					color.Green("    %s: %s \n", evt.Reason, evt.Message)
				}
			}
			color.Cyan("------------------------------------------------------- \n")
		}
	}
	color.Cyan("----------------------------------------------- \n")
}

// monitorResources monitors the resources created by the Helm release until they are all ready.
// It checks the status of the resources in the Kubernetes API and waits until they are all ready.
// The function returns an error if the resources are not ready within the specified timeout.
func monitorResources(rel *release.Release, namespace string, timeout time.Duration) error {
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
			spinner.Fail("Error checking resources readiness \n")
			return fmt.Errorf("error checking resources readiness: %v \n", err)
		}
		if allReady {
			spinner.Success("All resources are ready. \n")
			return nil
		}
		if time.Now().After(deadline) {
			spinner.Fail("Timeout waiting for all resources to become ready \n")
			return fmt.Errorf("timeout waiting for all resources to become ready \n")
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
				return false, nil, err
			}
			if dep.Status.ReadyReplicas != *dep.Spec.Replicas {
				notReadyResources = append(notReadyResources, fmt.Sprintf("Deployment/%s (%d/%d)", res.Name, dep.Status.ReadyReplicas, *dep.Spec.Replicas))
			}
		case "Pod":
			pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), res.Name, metav1.GetOptions{})
			if err != nil {
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
	return false, notReadyResources, nil
}

// describeFailedResources fetches detailed information about the failed resources in the Helm release.
// It retrieves the pods associated with the release and prints their status and events for troubleshooting.
// This function is used to provide additional context to the user when resources fail to be created or updated.
func describeFailedResources(namespace, releaseName string) {
	color.Cyan("----- TROUBLESHOOTING FAILED RESOURCES ----- \n")
	clientset, err := getKubeClient()
	if err != nil {
		color.Red("Error getting kube client: %v \n", err)
		return
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		color.Red("Failed to list pods for troubleshooting: %v \n", err)
		return
	}

	if len(podList.Items) == 0 {
		color.Yellow("No pods found for release '%s', cannot diagnose further.\n", releaseName)
		return
	}

	for _, pod := range podList.Items {
		color.Green("Pod: %s \n", pod.Name)
		color.Green("Phase: %s \n", pod.Status.Phase)
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				color.Red("Container: %s is waiting with reason: %s, message: %s \n", cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message)
			} else if cs.State.Terminated != nil {
				color.Red("Container: %s is terminated with reason: %s, message: %s \n", cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.Message)
			}
		}

		evts, err := clientset.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("involvedObject.name=%s", pod.Name),
		})
		if err != nil {
			color.Yellow("Error fetching events for pod %s: %v \n", pod.Name, err)
			continue
		}

		if len(evts.Items) == 0 {
			color.Yellow("No events found for pod %s \n", pod.Name)
		} else {
			color.Green("Events for pod %s: \n", pod.Name)
			for _, evt := range evts.Items {
				color.Green("  %s: %s \n", evt.Reason, evt.Message)
			}
		}
		color.Cyan("------------------------------------------------------- \n")
	}
	color.Cyan("----------------------------------------------- \n")
}

// resourceRemoved checks if the specified resource has been removed from the Kubernetes API.
// It returns true if the resource is not found, indicating that it has been removed.
// This function is used to determine if a resource has been successfully deleted.
func resourceRemoved(clientset *kubernetes.Clientset, namespace string, r Resource) bool {
	switch r.Kind {
	case "Deployment":
		_, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "Pod":
		_, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "Service":
		_, err := clientset.CoreV1().Services(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "ServiceAccount":
		_, err := clientset.CoreV1().ServiceAccounts(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "ReplicaSet":
		_, err := clientset.AppsV1().ReplicaSets(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "StatefulSet":
		_, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "DaemonSet":
		_, err := clientset.AppsV1().DaemonSets(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "ConfigMap":
		_, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "Secret":
		_, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	case "PersistentVolumeClaim":
		_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
		return isNotFound(err)
	default:
		return true
	}
}

// isNotFound checks if the error is a "not found" error.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}
