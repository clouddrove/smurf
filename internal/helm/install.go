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
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func HelmInstall(
	releaseName, chartRef, namespace string,
	valuesFiles []string,
	timeout time.Duration,
	atomic, debug bool,
	setValues, setLiteralValues []string,
	repoURL, version string,
) error {
	if debug {
		pterm.EnableDebugMessages()
		pterm.Debug.Println("========== HELM DEBUG MODE ==========")
		pterm.Debug.Printfln("Install Configuration:")
		pterm.Debug.Printfln("  Release Name: %s", releaseName)
		pterm.Debug.Printfln("  Chart Reference: %s", chartRef)
		pterm.Debug.Printfln("  Namespace: %s", namespace)
		pterm.Debug.Printfln("  Timeout: %v", timeout)
		pterm.Debug.Printfln("  Atomic: %v", atomic)
		pterm.Debug.Printfln("  Repo URL: %s", repoURL)
		pterm.Debug.Printfln("  Version: %s", version)
		pterm.Debug.Println("====================================")
	}

	// Namespace handling
	if debug {
		pterm.Debug.Println("Checking/creating namespace...")
	}
	if err := ensureNamespace(namespace, true, debug); err != nil {
		return fmt.Errorf("namespace setup failed: %w", err)
	}

	// Helm initialization
	settings := cli.New()
	settings.SetNamespace(namespace)
	settings.Debug = debug

	actionConfig := new(action.Configuration)
	logFn := func(format string, v ...interface{}) {
		if debug {
			pterm.Debug.Printf(format, v...)
		}
	}

	if debug {
		pterm.Debug.Println("Initializing Helm configuration...")
	}
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logFn); err != nil {
		return fmt.Errorf("helm initialization failed: %w", err)
	}

	// Chart loading
	if debug {
		pterm.Debug.Println("Loading chart...")
	}
	chartObj, err := LoadChart(chartRef, repoURL, version, settings)
	if err != nil {
		return fmt.Errorf("chart loading failed: %w", err)
	}

	if debug {
		pterm.Debug.Printfln("Chart Details:")
		pterm.Debug.Printfln("  Name: %s", chartObj.Metadata.Name)
		pterm.Debug.Printfln("  Version: %s", chartObj.Metadata.Version)
		pterm.Debug.Printfln("  Description: %s", chartObj.Metadata.Description)
		pterm.Debug.Printfln("  Dependencies: %d", len(chartObj.Metadata.Dependencies))
	}

	// Values processing
	if debug {
		pterm.Debug.Println("Processing values...")
	}
	vals, err := loadAndMergeValuesWithSetsInstall(valuesFiles, setValues, setLiteralValues, debug)
	if err != nil {
		return fmt.Errorf("values processing failed: %w", err)
	}

	// Installation
	client := action.NewInstall(actionConfig)
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Atomic = atomic
	client.Wait = true
	client.Timeout = timeout
	client.CreateNamespace = true

	if debug {
		pterm.Debug.Println("Running Helm install...")
	}
	rel, err := client.Run(chartObj, vals)
	if err != nil {
		if debug {
			pterm.Debug.Printfln("Installation failed with error: %v", err)
		}
		return fmt.Errorf("installation failed: %w", err)
	}

	// Post-installation output
	if debug {
		printReleaseInfo(rel, true)
		printResourcesFromRelease(rel)
		pterm.Debug.Println("Monitoring resources...")
	} else {
		printReleaseInfo(rel, false)
		printResourcesFromRelease(rel)
	}

	if err := monitorResources(rel, namespace, timeout, debug); err != nil {
		return fmt.Errorf("resource monitoring failed: %w", err)
	}

	if debug {
		pterm.Debug.Println("========== INSTALLATION COMPLETE ==========")
	}
	return nil
}

func LoadChart(chartRef, repoURL, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	if repoURL != "" {
		if settings.Debug {
			pterm.Debug.Printfln("Loading remote chart from repository: %s", repoURL)
		}
		return LoadRemoteChart(chartRef, repoURL, version, settings)
	}

	if strings.Contains(chartRef, "/") && !strings.HasPrefix(chartRef, ".") && !filepath.IsAbs(chartRef) {
		if settings.Debug {
			pterm.Debug.Printfln("Loading chart from local repository reference: %s", chartRef)
		}
		return LoadFromLocalRepo(chartRef, version, settings)
	}

	if settings.Debug {
		pterm.Debug.Printfln("Loading local chart from path: %s", chartRef)
	}
	return loader.Load(chartRef)
}

func LoadFromLocalRepo(chartRef, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	parts := strings.Split(chartRef, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid chart reference format, expected 'repo/chart'")
	}
	repoName, chartName := parts[0], parts[1]

	repoFile, err := repo.LoadFile(settings.RepositoryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load repositories file: %w", err)
	}

	var repoURL string
	for _, r := range repoFile.Repositories {
		if r.Name == repoName {
			repoURL = r.URL
			break
		}
	}

	if repoURL == "" {
		return nil, fmt.Errorf("repository %s not found", repoName)
	}

	return LoadRemoteChart(chartName, repoURL, version, settings)
}

func LoadRemoteChart(chartName, repoURL, version string, settings *cli.EnvSettings) (*chart.Chart, error) {
	repoEntry := &repo.Entry{
		Name: "temp-repo",
		URL:  repoURL,
	}

	chartRepo, err := repo.NewChartRepository(repoEntry, getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to create chart repository: %w", err)
	}

	if settings.Debug {
		pterm.Debug.Printfln("Downloading repository index from %s", repoURL)
	}
	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return nil, fmt.Errorf("failed to download repository index: %w", err)
	}

	chartURL, err := repo.FindChartInRepoURL(repoURL, chartName, version, "", "", "", getter.All(settings))
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart in repository: %w", err)
	}

	chartDownloader := downloader.ChartDownloader{
		Out:     os.Stdout,
		Getters: getter.All(settings),
		Options: []getter.Option{},
	}

	if settings.Debug {
		pterm.Debug.Printfln("Downloading chart from %s", chartURL)
	}
	chartPath, _, err := chartDownloader.DownloadTo(chartURL, version, settings.RepositoryCache)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart: %w", err)
	}

	return loader.Load(chartPath)
}

func loadAndMergeValuesWithSetsInstall(valuesFiles []string, setValues, setLiteralValues []string, debug bool) (map[string]interface{}, error) {
	if debug {
		pterm.Debug.Printfln("Merging values from %d files", len(valuesFiles))
		for _, v := range valuesFiles {
			pterm.Debug.Printfln("  - %s", v)
		}
		if len(setValues) > 0 {
			pterm.Debug.Printfln("Applying --set values:")
			for _, v := range setValues {
				pterm.Debug.Printfln("  - %s", v)
			}
		}
		if len(setLiteralValues) > 0 {
			pterm.Debug.Printfln("Applying --set-literal values:")
			for _, v := range setLiteralValues {
				pterm.Debug.Printfln("  - %s", v)
			}
		}
	}

	valueOpts := &values.Options{
		ValueFiles:    valuesFiles,
		StringValues:  setValues,
		LiteralValues: setLiteralValues,
	}

	return valueOpts.MergeValues(getter.All(cli.New()))
}

func isResourceReady(clientset *kubernetes.Clientset, resource map[string]interface{}, namespace string, debug bool) (bool, error) {
	kind, ok := resource["kind"].(string)
	if !ok {
		return false, fmt.Errorf("missing kind in resource")
	}

	name, ok := resource["metadata"].(map[string]interface{})["name"].(string)
	if !ok {
		return false, fmt.Errorf("missing name in resource metadata")
	}

	if debug {
		pterm.Debug.Printfln("Checking resource: %s/%s", kind, name)
	}

	switch strings.ToLower(kind) {
	case "deployment":
		return isDeploymentReady(clientset, name, namespace, debug)
	case "statefulset":
		return isStatefulSetReady(clientset, name, namespace, debug)
	case "daemonset":
		return isDaemonSetReady(clientset, name, namespace, debug)
	case "service":
		return isServiceReady(clientset, name, namespace, debug)
	case "pod":
		return isPodReady(clientset, name, namespace, debug)
	case "replicaset":
		return isReplicaSetReady(clientset, name, namespace, debug)
	case "job":
		return isJobReady(clientset, name, namespace, debug)
	case "persistentvolumeclaim":
		return isPVCReady(clientset, name, namespace, debug)
	default:
		if debug {
			pterm.Debug.Printfln("Skipping resource check for %s (unsupported kind)", kind)
		}
		return true, nil // Skip unsupported resource types
	}
}

func isDeploymentReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get deployment %s: %w", name, err)
	}

	if deployment.Status.ObservedGeneration < deployment.Generation {
		if debug {
			pterm.Debug.Printfln("Deployment %s: observed generation %d < generation %d", name, deployment.Status.ObservedGeneration, deployment.Generation)
		}
		return false, nil
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			if debug {
				pterm.Debug.Printfln("Deployment %s: %d/%d replicas available", name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
			}
			return deployment.Status.AvailableReplicas == *deployment.Spec.Replicas, nil
		}
	}

	return false, nil
}

func isStatefulSetReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	statefulset, err := clientset.AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get statefulset %s: %w", name, err)
	}

	if statefulset.Status.ObservedGeneration < statefulset.Generation {
		return false, nil
	}

	if debug {
		pterm.Debug.Printfln("StatefulSet %s: %d/%d replicas ready", name, statefulset.Status.ReadyReplicas, *statefulset.Spec.Replicas)
	}
	return statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas, nil
}

func isDaemonSetReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	daemonset, err := clientset.AppsV1().DaemonSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get daemonset %s: %w", name, err)
	}

	if daemonset.Status.ObservedGeneration < daemonset.Generation {
		return false, nil
	}

	if debug {
		pterm.Debug.Printfln("DaemonSet %s: %d/%d pods ready", name, daemonset.Status.NumberReady, daemonset.Status.DesiredNumberScheduled)
	}
	return daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled, nil
}

func isServiceReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	_, err := clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get service %s: %w", name, err)
	}
	return true, nil // Services are always "ready" once created
}

func isPodReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get pod %s: %w", name, err)
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			if debug {
				pterm.Debug.Printfln("Pod %s: ready", name)
			}
			return true, nil
		}
	}

	if debug {
		pterm.Debug.Printfln("Pod %s: not ready", name)
	}
	return false, nil
}

func isReplicaSetReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	replicaset, err := clientset.AppsV1().ReplicaSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get replicaset %s: %w", name, err)
	}

	if debug {
		pterm.Debug.Printfln("ReplicaSet %s: %d/%d replicas ready", name, replicaset.Status.ReadyReplicas, replicaset.Status.Replicas)
	}
	return replicaset.Status.ReadyReplicas == replicaset.Status.Replicas, nil
}

func isJobReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	job, err := clientset.BatchV1().Jobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get job %s: %w", name, err)
	}

	if job.Status.Succeeded > 0 {
		if debug {
			pterm.Debug.Printfln("Job %s: completed successfully", name)
		}
		return true, nil
	}

	if job.Status.Failed > 0 {
		return false, fmt.Errorf("job %s failed", name)
	}

	if debug {
		pterm.Debug.Printfln("Job %s: still running", name)
	}
	return false, nil
}

func isPVCReady(clientset *kubernetes.Clientset, name, namespace string, debug bool) (bool, error) {
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to get PVC %s: %w", name, err)
	}

	if pvc.Status.Phase == corev1.ClaimBound {
		if debug {
			pterm.Debug.Printfln("PVC %s: bound", name)
		}
		return true, nil
	}

	if debug {
		pterm.Debug.Printfln("PVC %s: phase %s", name, pvc.Status.Phase)
	}
	return false, nil
}

func getResourceName(resource map[string]interface{}) string {
	kind, _ := resource["kind"].(string)
	name, _ := resource["metadata"].(map[string]interface{})["name"].(string)
	return fmt.Sprintf("%s/%s", kind, name)
}
