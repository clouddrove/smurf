package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	settings      = cli.New()
	kubeClientset *kubernetes.Clientset
)

func init() {
	if os.Getenv("KUBECONFIG") != "" {
		settings.KubeConfig = os.Getenv("KUBECONFIG")
	} else {
		home := homedir.HomeDir()
		settings.KubeConfig = filepath.Join(home, ".kube", "config")
	}
}

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

func CreateChart(chartName, saveDir string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Creating Helm chart '%s' in directory '%s'...", chartName, saveDir))
	defer spinner.Stop()

	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
			color.Red("Failed to create directory '%s': %v \n", saveDir, err)
			return err
		}
	}

	_, err := chartutil.Create(chartName, saveDir)
	if err != nil {
		color.Red("Failed to create chart '%s': %v \n", chartName, err)
		return err
	}
	homePathOfCreatedChart := filepath.Join(saveDir, chartName)
	spinner.Success(fmt.Sprintf("Chart '%s' created successfully at '%s'", chartName, homePathOfCreatedChart))
	return nil
}

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

func debugLog(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	fmt.Println()
}

func HelmInstall(releaseName, chartPath, namespace string, valuesFiles []string, duration time.Duration, Atomic bool, debug bool, setValues []string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Starting Helm Install for release: %s", releaseName))
	defer spinner.Stop()

	if err := ensureNamespace(namespace, true); err != nil {
		logDetailedError("namespace creation", err, namespace, releaseName)
		return err
	}

	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)

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

	client := action.NewInstall(actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Atomic = Atomic
	client.Wait = true
	client.Timeout = duration 
	client.CreateNamespace = true

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		color.Red("Chart Loading Failed: %s \n", chartPath)
		color.Red("Error: %v \n", err)
		color.Yellow("Try 'helm lint %s' to identify chart issues. \n", chartPath)
		return err
	}

	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues)
	if err != nil {
		logDetailedError("values loading", err, namespace, releaseName)
		return err
	}

	rel, err := client.Run(chartObj, vals)
	if err != nil {
		logDetailedError("helm install", err, namespace, releaseName)
		return err
	}

	if rel == nil { 
		err := fmt.Errorf("no release object returned by Helm")
		logDetailedError("release object", err, namespace, releaseName)
		return err
	}

	spinner.Success(fmt.Sprintf("Installation Completed Successfully for release: %s \n", releaseName))
	printReleaseInfo(rel)

	printResourcesFromRelease(rel)

	err = monitorResources(rel, namespace, client.Timeout)
	if err != nil {
		logDetailedError("resource monitoring", err, namespace, releaseName)
		return err
	}

	color.Green("All resources for release '%s' are ready and running.\n", releaseName)
	return nil
}

func HelmUpgrade(releaseName, chartPath, namespace string, setValues []string, valuesFiles []string, createNamespace, atomic bool, timeout time.Duration, debug bool) error {
	color.Green("Starting Helm Upgrade for release: %s \n", releaseName)

	if createNamespace {
		if err := ensureNamespace(namespace, true); err != nil {
			logDetailedError("namespace creation", err, namespace, releaseName)
			return err
		}
	}

	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)


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

	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Atomic = atomic
	client.Timeout = timeout
	client.Wait = true 

	chartObj, err := loader.Load(chartPath)
	if err != nil {
		color.Red("Chart Loading Failed: %s \n", chartPath)
		color.Red("Error: %v \n", err)
		color.Yellow("Try 'helm lint %s' to identify chart issues. \n", chartPath)
		return err
	}

	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues)
	if err != nil {
		logDetailedError("values loading", err, namespace, releaseName)
		return err
	}

	rel, err := client.Run(releaseName, chartObj, vals)
	if err != nil {
		logDetailedError("helm upgrade", err, namespace, releaseName)
		return err
	}

	if rel == nil {
		err := fmt.Errorf("no release object returned")
		logDetailedError("release object", err, namespace, releaseName)
		return err
	}

	color.Green("Upgrade Completed Successfully for release: %s \n", releaseName)
	printReleaseInfo(rel)
	printResourcesFromRelease(rel)

	color.Green("All resources for release '%s' after upgrade are ready and running.\n", releaseName)
	return nil
}

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

func loadAndMergeValuesWithSets(valuesFiles, setValues []string) (map[string]interface{}, error) {
	vals, err := loadAndMergeValues(valuesFiles)
	if err != nil {
		return nil, err
	}

	for _, set := range setValues {
		color.Green("Parsing set value: %s \n", set)
		if err := strvals.ParseInto(set, vals); err != nil {
			color.Red("Error parsing set value '%s': %v \n", set, err)
			return nil, err
		}
	}
	return vals, nil
}

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

type Resource struct {
	Kind string
	Name string
}

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

func HelmList(namespace string) ([]*release.Release, error) {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Listing releases in namespace: %s", namespace))
	defer spinner.Stop()

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		color.Red("Failed to initialize action configuration: %v \n", err)
		return nil, err
	}

	client := action.NewList(actionConfig)
	client.AllNamespaces = true

	releases, err := client.Run()
	if err != nil {
		color.Red("Failed to list releases: %v \n", err)
		return nil, err
	}

	fmt.Println()
	color.Cyan("%-17s %-10s %-8s %-20s %-7s %-30s \n", "NAME", "NAMESPACE", "REVISION", "UPDATED", "STATUS", "CHART")
	for _, rel := range releases {
		updatedStr := rel.Info.LastDeployed.Local().Format("2006-01-02 15:04:05")
		color.Yellow("%-17s %-10s %-8d %-20s %-7s %-30s \n",
			rel.Name, rel.Namespace, rel.Version, updatedStr, rel.Info.Status.String(), rel.Chart.Metadata.Name+"-"+rel.Chart.Metadata.Version)
	}

	spinner.Success("Releases listed successfully. \n")
	return releases, nil
}

func HelmUninstall(releaseName, namespace string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Starting Helm Uninstall for release: %s", releaseName))
	defer spinner.Stop()

	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), debugLog); err != nil {
		logDetailedError("helm uninstall", err, namespace, releaseName)
		return err
	}

	statusAction := action.NewStatus(actionConfig)
	rel, preErr := statusAction.Run(releaseName)

	if preErr == nil && rel != nil {
		printResourcesFromRelease(rel)
	} else {
		color.Yellow("Could not retrieve release '%s' status before uninstall: %v \n", releaseName, preErr)
	}

	client := action.NewUninstall(actionConfig)
	if client == nil {
		err := fmt.Errorf("failed to create Helm uninstall client")
		logDetailedError("helm uninstall", err, namespace, releaseName)
		return err
	}

	resp, err := client.Run(releaseName)
	if err != nil {
		logDetailedError("helm uninstall", err, namespace, releaseName)
		return err
	}

	color.Green("Uninstall Completed Successfully for release: %s \n", releaseName)

	var resources []Resource
	if len(resources) == 0 && resp != nil && resp.Release != nil {
		rs, err := parseResourcesFromManifest(resp.Release.Manifest)
		if err == nil {
			resources = rs
		} else {
			color.Yellow("Could not parse manifest from uninstall response for release '%s': %v \n", releaseName, err)
		}
	}

	if resp != nil && resp.Release != nil {
		color.Cyan("Detailed Information After Uninstall: \n")
		printResourcesFromRelease(resp.Release)
	}

	if len(resources) > 0 {
		color.Cyan("----- RESOURCES REMOVED ----- \n")
		clientset, getErr := getKubeClient()
		if getErr != nil {
			color.Yellow("Could not verify resource removal due to kubeclient error: %v \n", getErr)
			for _, r := range resources {
				color.Green("%s: %s (Assumed Removed) \n", r.Kind, r.Name)
			}
		} else {
			for _, r := range resources {
				removed := resourceRemoved(clientset, namespace, r)
				if removed {
					color.Green("%s: %s (Removed) \n", r.Kind, r.Name)
				} else {
					color.Yellow("%s: %s might still exist. Check your cluster. \n", r.Kind, r.Name)
				}
			}
		}
		color.Cyan("-------------------------------- \n")
	} else {
		color.Green("No resources recorded for this release or unable to parse manifest. Assuming all are removed. \n")
	}

	color.Green("All resources associated with release '%s' have been removed (or no longer found). \n", releaseName)
	return nil
}

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

func isNotFound(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not found")
}

func HelmLint(chartPath string, fileValues []string) error {
	spinner, _ := pterm.DefaultSpinner.Start("Linting chart")
	defer spinner.Stop()

	client := action.NewLint()

	vals := make(map[string]interface{})
	for _, f := range fileValues {
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			color.Red("Failed to read values file '%s': %v \n", f, err)
			return err
		}
		for key, value := range additionalVals {
			vals[key] = value
		}
	}

	for _, set := range fileValues {
		if err := strvals.ParseInto(set, vals); err != nil {
			color.Red("Failed to parse set values '%s': %v \n", set, err)
			return err
		}
	}

	result := client.Run([]string{chartPath}, vals)
	if len(result.Messages) > 0 {
		for _, msg := range result.Messages {
			color.Yellow("Severity: %s \n", msg.Severity)
			color.Yellow("Path: %s \n", msg.Path)
			fmt.Println(msg)
			fmt.Println()
		}
		spinner.Fail("Linting issues found \n")
	} else {
		color.Green("No linting issues found in the chart %s \n", chartPath)
		spinner.Success("Linting completed successfully \n")
	}
	return nil
}

func HelmTemplate(releaseName, chartPath, namespace string, valuesFiles []string) error {
	settings := cli.New()
	actionConfig := new(action.Configuration)

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		color.Red("Failed to initialize action configuration: %v \n", err)
		return err
	}

	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Replace = true
	client.ClientOnly = true

	chart, err := loader.Load(chartPath)
	if err != nil {
		color.Red("Failed to load chart '%s': %v \n", chartPath, err)
		return err
	}

	vals := make(map[string]interface{})
	for _, f := range valuesFiles {
		additionalVals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			color.Red("Error reading values file '%s': %v \n", f, err)
			return err
		}
		for key, value := range additionalVals {
			vals[key] = value
		}
	}

	for _, set := range valuesFiles {
		if err := strvals.ParseInto(set, vals); err != nil {
			color.Red("Error parsing set values '%s': %v \n", set, err)
			return err
		}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Rendering Helm templates...\n")
	rel, err := client.Run(chart, vals)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to render templates: %v \n", err))
		return err
	}
	spinner.Success("Templates rendered successfully \n")

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Println(green(rel.Manifest))

	return nil
}

func HelmProvision(releaseName, chartPath, namespace string) error {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), nil); err != nil {
		color.Red("Failed to initialize Helm action configuration: %v \n", err)
		return err
	}

	client := action.NewList(actionConfig)
	results, err := client.Run()
	if err != nil {
		color.Red("Failed to list releases: %v \n", err)
		return err
	}

	var wg sync.WaitGroup
	var installErr, upgradeErr, lintErr, templateErr error

	exists := false
	for _, result := range results {
		if result.Name == releaseName {
			exists = true
			break
		}
	}

	wg.Add(1)
	if exists {
		go func() {
			defer wg.Done()
			upgradeErr = HelmUpgrade(releaseName, chartPath, namespace, nil, nil, false, false, 0, false)
		}()
	} else {
		go func() {
			defer wg.Done()
			installErr = HelmInstall(releaseName, chartPath, namespace, nil, 300, false, false, []string{})
		}()
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		lintErr = HelmLint(chartPath, nil)
	}()

	go func() {
		defer wg.Done()
		templateErr = HelmTemplate(releaseName, chartPath, namespace, nil)
	}()

	wg.Wait()

	if installErr != nil || upgradeErr != nil || lintErr != nil || templateErr != nil {
		if installErr != nil {
			color.Red("Install failed: %v \n", installErr)
		}
		if upgradeErr != nil {
			color.Red("Upgrade failed: %v \n", upgradeErr)
		}
		if lintErr != nil {
			color.Red("Lint failed: %v \n", lintErr)
		}
		if templateErr != nil {
			color.Red("Template rendering failed: %v \n", templateErr)
		}
		return fmt.Errorf("provisioning failed \n")
	}

	color.Green("Provisioning completed successfully. \n")
	return nil
}

func HelmReleaseExists(releaseName, namespace string) (bool, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		return false, err
	}

	list := action.NewList(actionConfig)
	list.Deployed = true
	list.AllNamespaces = false
	releases, err := list.Run()
	if err != nil {
		return false, err
	}

	for _, rel := range releases {
		if rel.Name == releaseName && rel.Namespace == namespace {
			return true, nil
		}
	}

	return false, nil
}

func HelmStatus(releaseName, namespace string) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Retrieving status for release: %s", releaseName))
	defer spinner.Stop()

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, "secrets", nil); err != nil {
		logDetailedError("helm status", err, namespace, releaseName)
		return err
	}

	statusAction := action.NewStatus(actionConfig)
	rel, err := statusAction.Run(releaseName)
	if err != nil {
		logDetailedError("helm status", err, namespace, releaseName)
		return err
	}

	data := [][]string{
		{"NAME", rel.Name},
		{"NAMESPACE", rel.Namespace},
		{"STATUS", rel.Info.Status.String()},
		{"REVISION", fmt.Sprintf("%d", rel.Version)},
		{"TEST SUITE", "None"},
	}

	pterm.DefaultTable.WithHasHeader(false).WithData(data).Render()

	if rel.Info.Notes != "" {
		color.Green("NOTES: %s\n", rel.Info.Notes)
	} else {
		color.Yellow("No additional notes provided for this release.\n")
	}

	printResourcesFromRelease(rel)

	resources, err := parseResourcesFromManifest(rel.Manifest)
	if err != nil {
		color.Red("Error parsing manifest for readiness check: %v \n", err)
		return nil
	}

	clientset, err := getKubeClient()
	if err != nil {
		color.Red("Error getting kube client for readiness check: %v \n", err)
		return err
	}

	allReady, notReadyResources, err := resourcesReady(clientset, rel.Namespace, resources)
	if err != nil {
		color.Red("Error checking resource readiness: %v \n", err)
		return err
	}

	if !allReady {
		color.Yellow("Some resources are not ready: \n")
		for _, nr := range notReadyResources {
			color.Yellow("- %s \n", nr)
		}
		describeFailedResources(rel.Namespace, rel.Name)
	} else {
		color.Green("All resources for release '%s' are ready.\n", rel.Name)
	}

	spinner.Success(fmt.Sprintf("Status retrieved successfully for release: %s \n", releaseName))
	return nil
}

type RollbackOptions struct {
	Namespace string
	Debug     bool
	Force     bool
	Timeout   int
	Wait      bool
}

func HelmRollback(releaseName string, revision int, opts RollbackOptions) error {
	if releaseName == "" {
		return fmt.Errorf("release name cannot be empty \n")
	}
	if revision < 1 {
		return fmt.Errorf("revision must be a positive integer \n")
	}

	color.Green("Starting Helm Rollback for release: %s to revision %d \n", releaseName, revision)

	settings := cli.New()
	settings.Debug = opts.Debug

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), opts.Namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		if settings.Debug {
			fmt.Printf(format, v...)
		}
	}); err != nil {
		logDetailedError("helm rollback", err, opts.Namespace, releaseName)
		return fmt.Errorf("failed to initialize Helm action configuration: %w \n", err)
	}

	rollbackAction := action.NewRollback(actionConfig)
	rollbackAction.Version = revision
	rollbackAction.Force = opts.Force
	rollbackAction.Timeout = time.Duration(opts.Timeout) * time.Second
	rollbackAction.Wait = opts.Wait

	if err := rollbackAction.Run(releaseName); err != nil {
		logDetailedError("helm rollback", err, opts.Namespace, releaseName)
		return err
	}

	if err := HelmStatus(releaseName, opts.Namespace); err != nil {
		color.Yellow("Rollback completed, but status retrieval failed. Check the release status manually.\n")
		return nil
	}

	color.Green("Rollback Completed Successfully for release: %s to revision %d \n", releaseName, revision)
	return nil
}
