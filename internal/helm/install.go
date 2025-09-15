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
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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

// HelmInstall handles chart installation with simple logging
func HelmInstall(
	releaseName, chartRef, namespace string, valuesFiles []string,
	duration time.Duration, atomic, debug bool,
	setValues, setLiteralValues []string, repoURL, version string,
) error {
	if err := ensureNamespace(namespace, true); err != nil {
		logDetailedError("namespace creation", err, namespace, releaseName)
		return err
	}

	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig := new(action.Configuration)

	logFn := func(format string, v ...interface{}) {
		if debug {
			fmt.Printf("🔍 "+format+"\n", v...)
		}
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logFn); err != nil {
		return fmt.Errorf("❌ Helm configuration failed: %w", err)
	}
	client := action.NewInstall(actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Atomic = atomic
	client.Wait = true
	client.Timeout = duration
	client.CreateNamespace = true

	var chartObj *chart.Chart
	var err error

	chartObj, err = LoadChart(chartRef, repoURL, version, settings)
	if err != nil {
		return fmt.Errorf("❌ Chart loading failed: %w", err)
	}

	// Load and merge values
	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteralValues, debug)
	if err != nil {
		return fmt.Errorf("❌ Values processing failed: %w", err)
	}

	rel, err := client.Run(chartObj, vals)
	if err != nil {
		errorLock(err)
		return fmt.Errorf("❌ Helm install failed: %w", err)
	}

	// Monitor resources and get detailed information
	err = monitorEssentialResources(rel, namespace, duration)
	if err != nil {
		return fmt.Errorf("❌ Resource monitoring failed: %w", err)
	}

	return nil
}

func errorLock(err error) {
	fmt.Println(pterm.Red("❌ Installation failed..."))
	fmt.Println(pterm.Red("Error : ", err))
}

// monitorEssentialResources monitors resources and prints detailed information
func monitorEssentialResources(rel *release.Release, namespace string, timeout time.Duration) error {
	details := &ResourceDetails{}

	clientset, err := getKubernetesClient()
	if err != nil {
		return err
	}

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
	details.Deployments, _ = getDetailedDeployments(clientset, namespace, rel.Name)
	details.ReplicaSets, _ = getDetailedReplicaSets(clientset, namespace, rel.Name)
	details.Pods, _ = getDetailedPods(clientset, namespace, rel.Name)
	details.Services, _ = getDetailedServices(clientset, namespace, rel.Name)
	details.Ingresses, _ = getDetailedIngresses(clientset, namespace, rel.Name)
	details.Secrets, _ = getDetailedSecrets(clientset, namespace, rel.Name)
	details.ConfigMaps, _ = getDetailedConfigMaps(clientset, namespace, rel.Name)

	// Print detailed resource summary
	printResourceSummaryHorizontal(details)

	return nil
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

					// Print table headers with proper spacing
					fmt.Printf("            %-40s%-15s%-10s%-15s\n", "NAME", "STATUS", "READY", "NODE")

					for _, pod := range deploymentPods {
						// Format the pod information in a table row
						fmt.Printf("            %-40s%-15s%-10s%-15s\n", pod.Name, pod.Status, pod.Ready, pod.Node)
					}
				}
			} else {
				fmt.Printf("    ├── %s%s%s\n", Yellow, dep.Name, Reset)

				// Show pods for this deployment in tabular format
				if len(deploymentPods) > 0 {
					fmt.Printf("    │   └── %sPODS%s\n", Magenta, Reset)

					// Print table headers with proper spacing
					fmt.Printf("    │       %-40s%-15s%-10s%-15s\n", "NAME", "STATUS", "READY", "NODE")

					for _, pod := range deploymentPods {
						// Format the pod information in a table row
						fmt.Printf("    │       %-40s%-15s%-10s%-15s\n", pod.Name, pod.Status, pod.Ready, pod.Node)
					}
				}
			}
		}
	}

	// ReplicaSets
	// if len(details.ReplicaSets) > 0 {
	// 	fmt.Println("└── REPLICASETS")
	// 	for i, rs := range details.ReplicaSets {
	// 		if i == len(details.ReplicaSets)-1 {
	// 			fmt.Printf("    └── %s%s%s\n", Yellow, rs.Name, Reset)
	// 			fmt.Printf("        └── %sReplicas:%s %d/%d/%d (Desired/Ready/Available)\n",
	// 				Cyan, Reset, rs.DesiredReplicas, rs.ReadyReplicas, rs.AvailableReplicas)
	// 		} else {
	// 			fmt.Printf("    ├── %s%s%s\n", Yellow, rs.Name, Reset)
	// 			fmt.Printf("    │   └── %sReplicas:%s %d/%d/%d (Desired/Ready/Available)\n",
	// 				Cyan, Reset, rs.DesiredReplicas, rs.ReadyReplicas, rs.AvailableReplicas)
	// 		}
	// 	}
	// }

	// Services
	if len(details.Services) > 0 {
		fmt.Println("└── SERVICES")
		fmt.Printf("     └── %-40s%-15s%-10s\n", "NAME", "TYPE", "PORTS")
		for i, svc := range details.Services {
			ports := strings.Join(svc.Ports, ", ")
			if ports == "" {
				ports = "No ports"
			}

			if i == len(details.Services)-1 {
				fmt.Printf("         %-40s%-15s%-10s\n", svc.Name, svc.Type, svc.Ports)
			} else {
				fmt.Printf("         %-40s%-15s%-10s\n", svc.Name, svc.Type, svc.Ports)
			}
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
				fmt.Printf("        └── %sType:%s %s\n", Cyan, Reset, secret.Type)
			} else {
				fmt.Printf("    ├── %s%s%s\n", Yellow, secret.Name, Reset)
				fmt.Printf("    │   └── %sType:%s %s\n", Cyan, Reset, secret.Type)
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

// Detailed resource getter functions (keep the same implementations as before)
func getDetailedDeployments(clientset *kubernetes.Clientset, namespace, releaseName string) ([]DeploymentInfo, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
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

func getDetailedReplicaSets(clientset *kubernetes.Clientset, namespace, releaseName string) ([]ReplicaSetInfo, error) {
	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{
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

func getDetailedPods(clientset *kubernetes.Clientset, namespace, releaseName string) ([]PodInfo, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
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

func getDetailedServices(clientset *kubernetes.Clientset, namespace, releaseName string) ([]ServiceInfo, error) {
	services, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{
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

func getDetailedIngresses(clientset *kubernetes.Clientset, namespace, releaseName string) ([]IngressInfo, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{
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

func getDetailedSecrets(clientset *kubernetes.Clientset, namespace, releaseName string) ([]SecretInfo, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{
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

func getDetailedConfigMaps(clientset *kubernetes.Clientset, namespace, releaseName string) ([]ConfigMapInfo, error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{
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

// getKubernetesClient creates a Kubernetes client
func getKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return clientset, nil
}

// LoadChart determines the chart source and loads it appropriately
func LoadChart(chartRef, repoURL, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	if repoURL != "" {
		return LoadRemoteChart(chartRef, repoURL, version, settings)
	}

	if strings.Contains(chartRef, "/") && !strings.HasPrefix(chartRef, ".") && !filepath.IsAbs(chartRef) {
		return LoadFromLocalRepo(chartRef, version, settings)
	}

	return loader.Load(chartRef)
}

// LoadFromLocalRepo loads a chart from a local repository
func LoadFromLocalRepo(chartRef, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	repoName := strings.Split(chartRef, "/")[0]
	chartName := strings.Split(chartRef, "/")[1]

	repoFile, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %v", err)
	}

	repoURL := ""
	for _, r := range repoFile.Repositories {
		if r.Name == repoName {
			repoURL = r.URL
			break
		}
	}

	if repoURL == "" {
		return nil, fmt.Errorf("repository %s not found in local repositories", repoName)
	}

	return LoadRemoteChart(chartName, repoURL, version, settings)
}

// LoadRemoteChart downloads and loads a chart from a remote repository
func LoadRemoteChart(chartName, repoURL string, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	repoEntry := &repo.Entry{
		Name: "temp-repo",
		URL:  repoURL,
	}

	chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to create chart repository: %v", err)
	}

	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return nil, fmt.Errorf("failed to download index file: %v", err)
	}

	chartURL, err := repo.FindChartInRepoURL(repoURL, chartName, version, "", "", "", getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to find chart in repository: %v", err)
	}

	chartDownloader := downloader.ChartDownloader{
		Out:     os.Stdout,
		Getters: getter.All(settings),
		Options: []getter.Option{},
	}

	chartPath, _, err := chartDownloader.DownloadTo(chartURL, version, settings.RepositoryCache)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart: %v", err)
	}

	return loader.Load(chartPath)
}
