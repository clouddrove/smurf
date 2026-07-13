package helm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ResourceDetails stores detailed information about resources
type ResourceDetails struct {
	ReleaseInfo *ReleaseInfo
	Deployments []DeploymentInfo
	ReplicaSets []ReplicaSetInfo
	Pods        []PodInfo
	Services    []ServiceInfo
	Ingresses   []IngressInfo
	Secrets     []SecretInfo
	ConfigMaps  []ConfigMapInfo
}

type ReleaseInfo struct {
	Name      string
	Chart     string
	Namespace string
	Status    string
	Revision  int
}

type DeploymentInfo struct {
	Name      string
	Replicas  int32
	Ready     int32
	Available int32
	UpToDate  int32
}

type ReplicaSetInfo struct {
	Name              string
	DesiredReplicas   int32
	ReadyReplicas     int32
	AvailableReplicas int32
}

type PodInfo struct {
	Name     string
	Ready    string
	Status   string
	Node     string
	Restarts int32
}

type ServiceInfo struct {
	Name      string
	Type      string
	ClusterIP string
	Ports     []string
}

type IngressInfo struct {
	Name    string
	Hosts   []string
	Address string
}

type SecretInfo struct {
	Name string
	Type string
}

type ConfigMapInfo struct {
	Name string
}

// ResourceChecker provides generic resource checking functionality
type ResourceChecker struct {
	clientset   *kubernetes.Clientset
	namespace   string
	releaseName string
	debug       bool
}

func NewResourceChecker(clientset *kubernetes.Clientset, namespace, releaseName string, debug bool) *ResourceChecker {
	return &ResourceChecker{
		clientset:   clientset,
		namespace:   namespace,
		releaseName: releaseName,
		debug:       debug,
	}
}

// getPodFailureReason extracts the detailed reason for pod failure
func getPodFailureReason(ctx context.Context, clientset *kubernetes.Clientset, pod *corev1.Pod) string {
	// Check container statuses first
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil {
			return stringFormat(status.State.Waiting.Reason, status.State.Waiting.Message)
		}
		if status.State.Terminated != nil {
			return fmt.Sprintf("%s: exit code %d - %s", status.State.Terminated.Reason, status.State.Terminated.ExitCode, status.State.Terminated.Message)
		}
	}

	// Try to get events for more context
	events, err := clientset.CoreV1().Events(pod.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", pod.Name),
	})
	if err == nil && len(events.Items) > 0 {
		// Get the most recent event
		latestEvent := events.Items[len(events.Items)-1]
		return stringFormat(latestEvent.Reason, latestEvent.Message)
	}

	return string(pod.Status.Phase)
}

func stringFormat(s1, s2 string) string {
	return fmt.Sprintf("%s: %s", s1, s2)
}

// isPodInFailureState checks if a pod is in a failure state that won't recover
func isPodInFailureState(pod *corev1.Pod) bool {
	// Check pod phase
	if pod.Status.Phase == corev1.PodFailed {
		return true
	}

	// Check container statuses for critical errors
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil {
			reason := status.State.Waiting.Reason
			// Critical errors that won't recover without intervention
			if reason == "ImagePullBackOff" || reason == "ErrImagePull" || reason == "CreateContainerConfigError" || reason == "InvalidImageName" || reason == "CrashLoopBackOff" || reason == "CreateContainerError" {
				return true
			}
		}
		if status.State.Terminated != nil {
			reason := status.State.Terminated.Reason
			if reason == "Error" || reason == "ContainerCannotRun" || status.State.Terminated.ExitCode != 0 {
				return true
			}
		}
	}

	// Check if pod has been stuck in Pending for too long with errors
	if pod.Status.Phase == corev1.PodPending {
		// If pod has been pending for more than 5 minutes and has container errors
		if time.Since(pod.CreationTimestamp.Time) > 5*time.Minute {
			for _, status := range pod.Status.ContainerStatuses {
				if status.State.Waiting != nil {
					reason := status.State.Waiting.Reason
					if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
						return true
					}
				}
			}
		}
	}

	return false
}

// handleInstallationSuccess handles successful installation
func handleInstallationSuccess(rel *release.Release, namespace string) error {
	if rel != nil {
		// Monitor resources and get detailed information
		fmt.Printf("👀 Monitoring resources...\n")
		err := monitorEssentialResources(rel, namespace)
		if err != nil {
			chartInfo := rel.Chart
			printErrorSummary("Resource Monitoring", rel.Name, namespace, chartInfo.Name(), err)
			return err
		}
	} else {
		fmt.Printf("✅ Installation completed successfully!\n")
	}
	return nil
}

// monitorEssentialResources monitors resources and prints detailed information
func monitorEssentialResources(rel *release.Release, namespace string) error {
	details := &ResourceDetails{}
	clientset, err := getKubeClient()
	if err != nil {
		return err
	}

	// Bound all the diagnostic List calls below with a single timeout so a slow
	// or unreachable API server can't hang this reporting path indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Print final summary
	fmt.Println()
	fmt.Println("🎉  Installation Summary")
	fmt.Println("------------------------")
	fmt.Printf("   Release Name:   %s\n", pterm.Green(rel.Name))
	fmt.Printf("   Namespace:      %s\n", pterm.Green(rel.Namespace))
	fmt.Printf("   Version:        %s\n", pterm.Green(fmt.Sprintf("%d", rel.Version)))
	fmt.Printf("   Status:         %s\n", pterm.Green(rel.Info.Status.String()))
	fmt.Printf("   Chart:          %s\n", pterm.Green(rel.Chart.Metadata.Name))
	fmt.Printf("   Chart Version:  %s\n", pterm.Green(rel.Chart.Metadata.Version))
	fmt.Println()

	// Set release info
	details.ReleaseInfo = &ReleaseInfo{
		Name:      rel.Name,
		Chart:     rel.Chart.Metadata.Name,
		Namespace: rel.Namespace,
		Status:    string(rel.Info.Status),
		Revision:  rel.Version,
	}

	// Get all resources
	details.Deployments, err = getDetailedDeployments(ctx, clientset, namespace, rel.Name)
	if err != nil {
		pterm.Warning.Printfln("Could not get deployment details: %v", err)
	}

	details.ReplicaSets, err = getDetailedReplicaSets(ctx, clientset, namespace, rel.Name)
	if err != nil {
		pterm.Warning.Printfln("Could not get replicaset details: %v", err)
	}

	details.Pods, err = getDetailedPods(ctx, clientset, namespace, rel.Name)
	if err != nil {
		pterm.Warning.Printfln("Could not get pod details: %v", err)
	}

	details.Services, err = getDetailedServices(ctx, clientset, namespace, rel.Name)
	if err != nil {
		pterm.Warning.Printfln("Could not get service details: %v", err)
	}

	details.Ingresses, err = getDetailedIngresses(ctx, clientset, namespace, rel.Name)
	if err != nil {
		pterm.Warning.Printfln("Could not get ingress details: %v", err)
	}

	details.Secrets, err = getDetailedSecrets(ctx, clientset, namespace, rel.Name)
	if err != nil {
		pterm.Warning.Printfln("Could not get secret details: %v", err)
	}

	details.ConfigMaps, err = getDetailedConfigMaps(ctx, clientset, namespace, rel.Name)
	if err != nil {
		pterm.Warning.Printfln("Could not get configmap details: %v", err)
	}

	// Print detailed resource summary
	printResourceSummaryHorizontal(details)
	return nil
}

// Detailed resource getter functions
func getDetailedDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace, releaseName string) ([]DeploymentInfo, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	var deploymentInfos []DeploymentInfo
	for _, dep := range deployments.Items {
		replicas := int32(0)
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}
		deploymentInfos = append(deploymentInfos, DeploymentInfo{
			Name:      dep.Name,
			Replicas:  replicas,
			Ready:     dep.Status.ReadyReplicas,
			Available: dep.Status.AvailableReplicas,
			UpToDate:  dep.Status.UpdatedReplicas,
		})
	}
	return deploymentInfos, nil
}

func getDetailedReplicaSets(ctx context.Context, clientset *kubernetes.Clientset, namespace, releaseName string) ([]ReplicaSetInfo, error) {
	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	var replicaSetInfos []ReplicaSetInfo
	for _, rs := range replicaSets.Items {
		replicas := int32(0)
		if rs.Spec.Replicas != nil {
			replicas = *rs.Spec.Replicas
		}
		replicaSetInfos = append(replicaSetInfos, ReplicaSetInfo{
			Name:              rs.Name,
			DesiredReplicas:   replicas,
			ReadyReplicas:     rs.Status.ReadyReplicas,
			AvailableReplicas: rs.Status.AvailableReplicas,
		})
	}
	return replicaSetInfos, nil
}

func getDetailedPods(ctx context.Context, clientset *kubernetes.Clientset, namespace, releaseName string) ([]PodInfo, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	var podInfos []PodInfo
	for _, pod := range pods.Items {
		ready := "0/0"
		totalContainers := len(pod.Spec.Containers)
		readyContainers := 0
		for _, status := range pod.Status.ContainerStatuses {
			if status.Ready {
				readyContainers++
			}
		}
		ready = fmt.Sprintf("%d/%d", readyContainers, totalContainers)

		restarts := int32(0)
		for _, status := range pod.Status.ContainerStatuses {
			restarts += status.RestartCount
		}

		nodeName := pod.Spec.NodeName
		if nodeName == "" {
			nodeName = "Not assigned"
		}

		podInfos = append(podInfos, PodInfo{
			Name:     pod.Name,
			Ready:    ready,
			Status:   string(pod.Status.Phase),
			Node:     nodeName,
			Restarts: restarts,
		})
	}
	return podInfos, nil
}

func getDetailedServices(ctx context.Context, clientset *kubernetes.Clientset, namespace, releaseName string) ([]ServiceInfo, error) {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	var serviceInfos []ServiceInfo
	for _, svc := range services.Items {
		var ports []string
		for _, port := range svc.Spec.Ports {
			portStr := fmt.Sprintf("%d", port.Port)
			if port.NodePort > 0 {
				portStr = fmt.Sprintf("%d:%d", port.Port, port.NodePort)
			}
			ports = append(ports, fmt.Sprintf("%s/%s", portStr, port.Protocol))
		}

		serviceInfos = append(serviceInfos, ServiceInfo{
			Name:      svc.Name,
			Type:      string(svc.Spec.Type),
			ClusterIP: string(svc.Spec.ClusterIP),
			Ports:     ports,
		})
	}
	return serviceInfos, nil
}

func getDetailedIngresses(ctx context.Context, clientset *kubernetes.Clientset, namespace, releaseName string) ([]IngressInfo, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	var ingressInfos []IngressInfo
	for _, ing := range ingresses.Items {
		var hosts []string
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
		}

		var addresses []string
		for _, addr := range ing.Status.LoadBalancer.Ingress {
			if addr.IP != "" {
				addresses = append(addresses, addr.IP)
			} else if addr.Hostname != "" {
				addresses = append(addresses, addr.Hostname)
			}
		}

		addressStr := strings.Join(addresses, ", ")
		if addressStr == "" {
			addressStr = "Pending"
		}

		ingressInfos = append(ingressInfos, IngressInfo{
			Name:    ing.Name,
			Hosts:   hosts,
			Address: addressStr,
		})
	}
	return ingressInfos, nil
}

func getDetailedSecrets(ctx context.Context, clientset *kubernetes.Clientset, namespace, releaseName string) ([]SecretInfo, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	var secretInfos []SecretInfo
	for _, secret := range secrets.Items {
		secretInfos = append(secretInfos, SecretInfo{
			Name: secret.Name,
			Type: string(secret.Type),
		})
	}
	return secretInfos, nil
}

func getDetailedConfigMaps(ctx context.Context, clientset *kubernetes.Clientset, namespace, releaseName string) ([]ConfigMapInfo, error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	var configMapInfos []ConfigMapInfo
	for _, cm := range configMaps.Items {
		configMapInfos = append(configMapInfos, ConfigMapInfo{
			Name: cm.Name,
		})
	}
	return configMapInfos, nil
}

func printResourceSummaryHorizontal(details *ResourceDetails) {
	// ANSI color codes
	const (
		Reset           = "\033[0m"
		Bold            = "\033[1m"
		Red             = "\033[31m"
		Green           = "\033[32m"
		Yellow          = "\033[33m"
		Blue            = "\033[34m"
		Magenta         = "\033[35m"
		Cyan            = "\033[36m"
		White           = "\033[37m"
		BlueBackground  = "\033[44m"
		GreenBackground = "\033[42m"
	)

	fmt.Printf("%s%s📁 RESOURCES%s%s\n", Bold, Green, Reset, Reset)

	// Helper function to find pods for a deployment
	getPodsForDeployment := func(deploymentName string) []PodInfo {
		var pods []PodInfo
		for _, pod := range details.Pods {
			// Try to match pods with deployment by name pattern
			if strings.HasPrefix(pod.Name, deploymentName) {
				pods = append(pods, pod)
			}
		}
		return pods
	}

	// Calculate dynamic column widths based on actual data (only NAME and STATUS now)
	calculatePodColumnWidths := func(pods []PodInfo) (int, int) {
		// Minimum widths
		maxPodNameWidth := 20
		maxStatusWidth := 8

		for _, pod := range pods {
			if len(pod.Name) > maxPodNameWidth {
				maxPodNameWidth = len(pod.Name)
			}
			if len(pod.Status) > maxStatusWidth {
				maxStatusWidth = len(pod.Status)
			}
		}

		// Add padding and set reasonable maximums
		maxPodNameWidth = min(maxPodNameWidth+2, 60)
		maxStatusWidth = min(maxStatusWidth+2, 20)

		return maxPodNameWidth, maxStatusWidth
	}

	// Calculate service column widths
	calculateServiceColumnWidths := func() (int, int) {
		maxServiceNameWidth := 20
		maxTypeWidth := 10

		for _, svc := range details.Services {
			if len(svc.Name) > maxServiceNameWidth {
				maxServiceNameWidth = len(svc.Name)
			}
			if len(svc.Type) > maxTypeWidth {
				maxTypeWidth = len(svc.Type)
			}
		}

		maxServiceNameWidth = min(maxServiceNameWidth+2, 50)
		maxTypeWidth = min(maxTypeWidth+2, 20)

		return maxServiceNameWidth, maxTypeWidth
	}

	// Deployments with their pods
	if len(details.Deployments) > 0 {
		fmt.Println("└── DEPLOYMENTS")
		for i, dep := range details.Deployments {
			deploymentPods := getPodsForDeployment(dep.Name)

			if i == len(details.Deployments)-1 {
				fmt.Printf("    └── %s%s%s\n", Yellow, dep.Name, Reset)

				// Show pods for this deployment in tabular format
				if len(deploymentPods) > 0 {
					fmt.Printf("        └── %sPODS%s\n", Magenta, Reset)

					// Calculate column widths for this specific deployment's pods
					podNameWidth, statusWidth := calculatePodColumnWidths(deploymentPods)

					// Print table headers with dynamic spacing (only NAME and STATUS)
					fmt.Printf("            %-*s %-*s\n",
						podNameWidth, "NAME",
						statusWidth, "STATUS")

					// Print separator line
					totalWidth := podNameWidth + statusWidth + 1
					fmt.Printf("            %s\n", strings.Repeat("─", totalWidth))

					for _, pod := range deploymentPods {
						// Format the pod information with dynamic column widths (only NAME and STATUS)
						fmt.Printf("            %-*s %-*s\n",
							podNameWidth, pod.Name,
							statusWidth, pod.Status)
					}
				}
			} else {
				fmt.Printf("    ├── %s%s%s\n", Yellow, dep.Name, Reset)

				// Show pods for this deployment in tabular format
				if len(deploymentPods) > 0 {
					fmt.Printf("    │   └── %sPODS%s\n", Magenta, Reset)

					// Calculate column widths for this specific deployment's pods
					podNameWidth, statusWidth := calculatePodColumnWidths(deploymentPods)

					// Print table headers with dynamic spacing (only NAME and STATUS)
					fmt.Printf("    │       %-*s %-*s\n",
						podNameWidth, "NAME",
						statusWidth, "STATUS")

					// Print separator line
					totalWidth := podNameWidth + statusWidth + 1
					fmt.Printf("    │       %s\n", strings.Repeat("─", totalWidth))

					for _, pod := range deploymentPods {
						// Format the pod information with dynamic column widths (only NAME and STATUS)
						fmt.Printf("    │       %-*s %-*s\n",
							podNameWidth, pod.Name,
							statusWidth, pod.Status)
					}
				}
			}
		}
	}

	// Services with dynamic column widths
	if len(details.Services) > 0 {
		fmt.Println("└── SERVICES")
		serviceNameWidth, typeWidth := calculateServiceColumnWidths()

		fmt.Printf("     └── %-*s %-*s %s\n", serviceNameWidth, "NAME", typeWidth, "TYPE", "PORTS")

		// Calculate total width for separator
		totalServiceWidth := serviceNameWidth + typeWidth + 20
		fmt.Printf("         %s\n", strings.Repeat("─", totalServiceWidth))

		for _, svc := range details.Services {
			ports := strings.Join(svc.Ports, ", ")
			if ports == "" {
				ports = "No ports"
			}

			fmt.Printf("         %-*s %-*s %s\n", serviceNameWidth, svc.Name, typeWidth, svc.Type, ports)
		}
	}

	// Ingresses
	if len(details.Ingresses) > 0 {
		fmt.Println("└── INGRESSES")
		for i, ing := range details.Ingresses {
			hosts := strings.Join(ing.Hosts, ", ")
			if hosts == "" {
				hosts = "No hosts"
			}
			if i == len(details.Ingresses)-1 {
				fmt.Printf("    └── %s%s%s\n", Yellow, ing.Name, Reset)
				fmt.Printf("        ├── %sHosts:%s %s\n", Cyan, Reset, hosts)
				fmt.Printf("        └── %sAddress:%s %s\n", Cyan, Reset, ing.Address)
			} else {
				fmt.Printf("    ├── %s%s%s\n", Yellow, ing.Name, Reset)
				fmt.Printf("    │   ├── %sHosts:%s %s\n", Cyan, Reset, hosts)
				fmt.Printf("    │   └── %sAddress:%s %s\n", Cyan, Reset, ing.Address)
			}
		}
	}

	// Secrets
	if len(details.Secrets) > 0 {
		fmt.Println("└── SECRETS")
		for i, secret := range details.Secrets {
			if i == len(details.Secrets)-1 {
				fmt.Printf("    └── %s%s%s\n", Yellow, secret.Name, Reset)
			} else {
				fmt.Printf("    ├── %s%s%s\n", Yellow, secret.Name, Reset)
			}
		}
	}

	// ConfigMaps
	if len(details.ConfigMaps) > 0 {
		fmt.Println("└── CONFIG MAPS")
		for i, cm := range details.ConfigMaps {
			if i == len(details.ConfigMaps)-1 {
				fmt.Printf("    └── %s%s%s\n", Yellow, cm.Name, Reset)
			} else {
				fmt.Printf("    ├── %s%s%s\n", Yellow, cm.Name, Reset)
			}
		}
	}
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Generic resource health checking functions
func (r *ResourceChecker) checkResourceHealth(
	ctx context.Context,
	resourceType string,
	listFunc func(context.Context) (interface{}, error),
	healthCheckFunc func(interface{}) (bool, error),
) (bool, error) {

	if r.debug {
		fmt.Printf("🔍 Checking %s\n", resourceType)
	}

	resources, err := listFunc(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list %s: %w", resourceType, err)
	}

	return healthCheckFunc(resources)
}

// Specific health check implementations
func (r *ResourceChecker) checkDeploymentsHealthy(ctx context.Context) (bool, error) {
	return r.checkResourceHealth(
		ctx,
		"deployments",
		func(ctx context.Context) (interface{}, error) {
			return r.clientset.AppsV1().Deployments(r.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", r.releaseName),
			})
		},
		func(resources interface{}) (bool, error) {
			deployments := resources.(*appsv1.DeploymentList)

			if len(deployments.Items) == 0 {
				if r.debug {
					fmt.Printf("🔍 No deployments found for release\n")
				}
				return true, nil
			}

			allHealthy := true
			for _, dep := range deployments.Items {
				if r.debug {
					fmt.Printf("🔍 Checking deployment %s: %d/%d replicas ready\n", dep.Name, dep.Status.ReadyReplicas, dep.Status.Replicas)
				}

				// Check if deployment is available
				if dep.Status.AvailableReplicas < dep.Status.Replicas {
					if r.debug {
						fmt.Printf("❌ Deployment %s not healthy: %d/%d replicas available\n", dep.Name, dep.Status.AvailableReplicas, dep.Status.Replicas)
					}
					allHealthy = false
				}

				// Check for deployment conditions that indicate failure
				for _, condition := range dep.Status.Conditions {
					if condition.Type == appsv1.DeploymentReplicaFailure && condition.Status == corev1.ConditionTrue {
						return false, fmt.Errorf("deployment %s has replica failure: %s", dep.Name, condition.Message)
					}
				}
			}
			return allHealthy, nil
		},
	)
}

func (r *ResourceChecker) checkStatefulSetsHealthy(ctx context.Context) (bool, error) {
	return r.checkResourceHealth(
		ctx,
		"statefulsets",
		func(ctx context.Context) (interface{}, error) {
			return r.clientset.AppsV1().StatefulSets(r.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", r.releaseName),
			})
		},
		func(resources interface{}) (bool, error) {
			statefulSets := resources.(*appsv1.StatefulSetList)

			if len(statefulSets.Items) == 0 {
				return true, nil
			}

			allHealthy := true
			for _, sts := range statefulSets.Items {
				if r.debug {
					fmt.Printf("🔍 Checking statefulset %s: %d/%d replicas ready\n", sts.Name, sts.Status.ReadyReplicas, sts.Status.Replicas)
				}

				if sts.Status.ReadyReplicas < sts.Status.Replicas {
					if r.debug {
						fmt.Printf("❌ StatefulSet %s not healthy: %d/%d replicas ready\n", sts.Name, sts.Status.ReadyReplicas, sts.Status.Replicas)
					}
					allHealthy = false
				}
			}
			return allHealthy, nil
		},
	)
}

func (r *ResourceChecker) checkDaemonSetsHealthy(ctx context.Context) (bool, error) {
	return r.checkResourceHealth(
		ctx,
		"daemonsets",
		func(ctx context.Context) (interface{}, error) {
			return r.clientset.AppsV1().DaemonSets(r.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", r.releaseName),
			})
		},
		func(resources interface{}) (bool, error) {
			daemonSets := resources.(*appsv1.DaemonSetList)

			if len(daemonSets.Items) == 0 {
				return true, nil
			}

			allHealthy := true
			for _, ds := range daemonSets.Items {
				if r.debug {
					fmt.Printf("🔍 Checking daemonset %s: %d/%d pods ready\n", ds.Name, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
				}

				if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
					if r.debug {
						fmt.Printf("❌ DaemonSet %s not healthy: %d/%d pods ready\n", ds.Name, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
					}
					allHealthy = false
				}
			}
			return allHealthy, nil
		},
	)
}

func (r *ResourceChecker) checkJobsHealthy(ctx context.Context) (bool, error) {
	return r.checkResourceHealth(
		ctx,
		"jobs",
		func(ctx context.Context) (interface{}, error) {
			return r.clientset.BatchV1().Jobs(r.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", r.releaseName),
			})
		},
		func(resources interface{}) (bool, error) {
			jobs := resources.(*batchv1.JobList)

			if len(jobs.Items) == 0 {
				return true, nil
			}

			for _, job := range jobs.Items {
				if r.debug {
					fmt.Printf("🔍 Checking job %s: %d succeeded, %d failed\n", job.Name, job.Status.Succeeded, job.Status.Failed)
				}

				// Job is considered failed if it has any failures
				if job.Status.Failed > 0 {
					return false, fmt.Errorf("job %s has failed: %d failures", job.Name, job.Status.Failed)
				}

				// Job is still running if no successes yet
				if job.Status.Succeeded == 0 {
					if r.debug {
						fmt.Printf("⏳ Job %s still running\n", job.Name)
					}
					return false, nil
				}
			}
			return true, nil
		},
	)
}

func (r *ResourceChecker) checkCronJobsHealthy(ctx context.Context) (bool, error) {
	return r.checkResourceHealth(
		ctx,
		"cronjobs",
		func(ctx context.Context) (interface{}, error) {
			return r.clientset.BatchV1().CronJobs(r.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", r.releaseName),
			})
		},
		func(resources interface{}) (bool, error) {
			cronJobs := resources.(*batchv1.CronJobList)

			if len(cronJobs.Items) == 0 {
				return true, nil
			}

			// For cronjobs, we just check that they exist and are scheduled properly
			// Actual job execution will be checked by the Jobs check above
			if r.debug {
				for _, cj := range cronJobs.Items {
					fmt.Printf("🔍 CronJob %s is scheduled\n", cj.Name)
				}
			}
			return true, nil
		},
	)
}

func (r *ResourceChecker) checkPodsHealthy(ctx context.Context) (bool, error) {
	return r.checkResourceHealth(
		ctx,
		"pods",
		func(ctx context.Context) (interface{}, error) {
			return r.clientset.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", r.releaseName),
			})
		},
		func(resources interface{}) (bool, error) {
			pods := resources.(*corev1.PodList)

			if len(pods.Items) == 0 {
				return true, nil
			}

			allHealthy := true
			for _, pod := range pods.Items {
				if r.debug {
					fmt.Printf("🔍 Checking pod %s: %s\n", pod.Name, pod.Status.Phase)
				}

				// Check for pod failures
				if isPodInFailureState(&pod) {
					failureReason := getPodFailureReason(ctx, r.clientset, &pod)
					return false, fmt.Errorf("pod %s failed: %s", pod.Name, failureReason)
				}

				// Check if pod is ready
				if !isPodReadyInstall(&pod) {
					if r.debug {
						fmt.Printf("❌ Pod %s not ready: %s\n", pod.Name, getPodReadyStatus(&pod))
					}
					allHealthy = false
				} else {
					if r.debug {
						fmt.Printf("✅ Pod %s is ready\n", pod.Name)
					}
				}
			}
			return allHealthy, nil
		},
	)
}

// verifyInstallationHealth checks if all resources are actually healthy after Helm reports success
func verifyInstallationHealth(namespace, releaseName string, timeout time.Duration, debug bool) error {
	clientset, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get kube client: %w", err)
	}

	checker := NewResourceChecker(clientset, namespace, releaseName, debug)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	maxWaitTime := 10 * time.Minute // Maximum wait time for resources to become healthy

	if debug {
		fmt.Printf("🔍 Starting comprehensive health verification for release '%s'\n", releaseName)
	}

	for {
		select {
		case <-ctx.Done():
			return checkFinalHealthStatus(clientset, namespace, releaseName, startTime, debug)
		case <-ticker.C:
			// Check all resource types
			allHealthy, err := checkAllResourcesHealthy(checker)
			if err != nil {
				return err
			}
			if allHealthy {
				if debug {
					fmt.Printf("✅ All resources are healthy!\n")
				}
				return nil
			}

			// Check if we've been waiting too long
			if time.Since(startTime) > maxWaitTime {
				return checkFinalHealthStatus(clientset, namespace, releaseName, startTime, debug)
			}

			if debug {
				fmt.Printf("🔍 Still waiting for resources to become healthy... (%v elapsed)\n", time.Since(startTime).Round(time.Second))
			}
		}
	}
}

// checkAllResourcesHealthy checks all resource types for health using the ResourceChecker.
// Each round of checks gets its own bounded timeout, independent of the overall
// polling deadline in verifyInstallationHealth, so a slow API server can't wedge a
// single tick forever.
func checkAllResourcesHealthy(checker *ResourceChecker) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check Deployments
	deploymentsHealthy, err := checker.checkDeploymentsHealthy(ctx)
	if err != nil {
		return false, err
	}
	if !deploymentsHealthy {
		return false, nil
	}

	// Check StatefulSets
	statefulSetsHealthy, err := checker.checkStatefulSetsHealthy(ctx)
	if err != nil {
		return false, err
	}
	if !statefulSetsHealthy {
		return false, nil
	}

	// Check DaemonSets
	daemonSetsHealthy, err := checker.checkDaemonSetsHealthy(ctx)
	if err != nil {
		return false, err
	}
	if !daemonSetsHealthy {
		return false, nil
	}

	// Check Jobs
	jobsHealthy, err := checker.checkJobsHealthy(ctx)
	if err != nil {
		return false, err
	}
	if !jobsHealthy {
		return false, nil
	}

	// Check CronJobs
	cronJobsHealthy, err := checker.checkCronJobsHealthy(ctx)
	if err != nil {
		return false, err
	}
	if !cronJobsHealthy {
		return false, nil
	}

	// Check Pods (as a final verification)
	podsHealthy, err := checker.checkPodsHealthy(ctx)
	if err != nil {
		return false, err
	}
	return podsHealthy, nil
}

// getPodReadyStatus returns a string describing the pod's ready status
func getPodReadyStatus(pod *corev1.Pod) string {
	readyContainers := 0
	totalContainers := len(pod.Spec.Containers)
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			readyContainers++
		}
	}

	status := fmt.Sprintf("%d/%d containers ready", readyContainers, totalContainers)

	// Add detailed container status if not all are ready
	if readyContainers < totalContainers {
		var containerStatuses []string
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				if cs.State.Waiting != nil {
					containerStatuses = append(containerStatuses, stringFormat(cs.Name, cs.State.Waiting.Reason))
				} else if cs.State.Terminated != nil {
					containerStatuses = append(containerStatuses, fmt.Sprintf("%s: %s (exit %d)", cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode))
				} else {
					containerStatuses = append(containerStatuses, fmt.Sprintf("%s: not ready", cs.Name))
				}
			}
		}
		if len(containerStatuses) > 0 {
			status += " - " + strings.Join(containerStatuses, ", ")
		}
	}

	return status
}

func isPodReadyInstall(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func checkFinalHealthStatus(clientset *kubernetes.Clientset, namespace, releaseName string, startTime time.Time, debug bool) error {
	var statusMessages []string

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get final status of all resources
	if deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	}); err == nil {
		for _, dep := range deployments.Items {
			statusMessages = append(statusMessages, fmt.Sprintf("Deployment %s: %d/%d ready", dep.Name, dep.Status.ReadyReplicas, dep.Status.Replicas))
		}
	}

	if pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	}); err == nil {
		for _, pod := range pods.Items {
			status := fmt.Sprintf("Pod %s: %s", pod.Name, pod.Status.Phase)
			if !isPodReadyInstall(&pod) {
				status += " [Not Ready]"
				for _, cs := range pod.Status.ContainerStatuses {
					if cs.State.Waiting != nil {
						status += fmt.Sprintf(" (%s)", cs.State.Waiting.Reason)
					}
				}
			}
			statusMessages = append(statusMessages, status)
		}
	}

	return fmt.Errorf("timeout: resources not healthy after %v. Status: %s",
		time.Since(startTime).Round(time.Second), strings.Join(statusMessages, "; "))
}
