package helm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HelmUpgrade(
	releaseName, chartRef, namespace string,
	setValues []string, valuesFiles []string, setLiteral []string,
	createNamespace, atomic bool, timeout time.Duration, debug bool,
	repoURL, version string,
) error {
	if debug {
		pterm.EnableDebugMessages()
		pterm.Debug.Println("========== HELM UPGRADE DEBUG MODE ==========")
		pterm.Debug.Printfln("Upgrade Configuration:")
		pterm.Debug.Printfln("  Release Name: %s", releaseName)
		pterm.Debug.Printfln("  Chart Reference: %s", chartRef)
		pterm.Debug.Printfln("  Namespace: %s", namespace)
		pterm.Debug.Printfln("  Timeout: %v", timeout)
		pterm.Debug.Printfln("  Atomic: %v", atomic)
		pterm.Debug.Printfln("  Create Namespace: %v", createNamespace)
		pterm.Debug.Printfln("  Repo URL: %s", repoURL)
		pterm.Debug.Printfln("  Version: %s", version)
		pterm.Debug.Println("============================================")
	}

	startTime := time.Now()

	// Handle namespace creation
	if createNamespace {
		if debug {
			pterm.Debug.Println("Ensuring namespace exists...")
		}
		if err := ensureNamespace(namespace, true, debug); err != nil {
			logDetailedError("namespace creation", err, namespace, releaseName)
			return fmt.Errorf("namespace setup failed: %w", err)
		}
	}

	// Initialize Helm configuration
	if debug {
		pterm.Debug.Println("Initializing Helm configuration...")
	}
	actionConfig, err := initActionConfig(namespace, debug)
	if err != nil {
		return fmt.Errorf("failed to initialize helm: %w", err)
	}

	// Verify release exists
	if debug {
		pterm.Debug.Println("Verifying release exists...")
	}
	if err := verifyReleaseExists(actionConfig, releaseName, namespace, debug); err != nil {
		return fmt.Errorf("release verification failed: %w", err)
	}

	// Load chart
	if debug {
		pterm.Debug.Println("Loading chart...")
	}
	chartObj, err := LoadChart(chartRef, repoURL, version, cli.New())
	if err != nil {
		return fmt.Errorf("failed to load chart: %w", err)
	}

	if debug {
		pterm.Debug.Printfln("Chart Details:")
		pterm.Debug.Printfln("  Name: %s", chartObj.Metadata.Name)
		pterm.Debug.Printfln("  Version: %s", chartObj.Metadata.Version)
		pterm.Debug.Printfln("  Description: %s", chartObj.Metadata.Description)
	}

	// Process values
	if debug {
		pterm.Debug.Println("Processing values...")
	}
	vals, err := loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteral, debug)
	if err != nil {
		return fmt.Errorf("failed to process values: %w", err)
	}

	// Setup upgrade client
	client := action.NewUpgrade(actionConfig)
	client.Namespace = namespace
	client.Atomic = atomic
	client.Timeout = timeout
	client.Wait = true
	client.WaitForJobs = true
	client.Force = false
	client.CleanupOnFail = atomic

	if debug {
		pterm.Debug.Printfln("Upgrade Client Configuration:")
		pterm.Debug.Printfln("  Namespace: %s", client.Namespace)
		pterm.Debug.Printfln("  Atomic: %v", client.Atomic)
		pterm.Debug.Printfln("  Timeout: %v", client.Timeout)
		pterm.Debug.Printfln("  Wait: %v", client.Wait)
		pterm.Debug.Printfln("  WaitForJobs: %v", client.WaitForJobs)
	}

	// Perform upgrade
	if debug {
		pterm.Debug.Println("Running Helm upgrade...")
	}
	rel, err := client.Run(releaseName, chartObj, vals)
	if err != nil {
		if debug {
			pterm.Debug.Printfln("Upgrade failed with error: %v", err)
			pterm.Debug.Println("Gathering debug information...")

			// Get detailed pod information for debugging
			pods, err := getPods(namespace, releaseName)
			if err == nil {
				pterm.Debug.Println("========== POD STATUS ==========")
				for _, pod := range pods {
					printPodDetails(pod, debug)
				}
			}
		}
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Monitor resources if atomic mode is enabled
	if atomic {
		if debug {
			pterm.Debug.Println("Monitoring resources (atomic mode)...")
		}
		if err := monitorResourcesUpgrade(rel, namespace, timeout, debug); err != nil {
			return fmt.Errorf("resource monitoring failed: %w", err)
		}
	}

	if debug {
		pterm.Debug.Printfln("Upgrade completed in %s", time.Since(startTime))
		printReleaseInfo(rel, true)
		printResourcesFromRelease(rel)
		pterm.Debug.Println("========== UPGRADE COMPLETE ==========")
	} else {
		printReleaseInfo(rel, false)
		printResourcesFromRelease(rel)
		pterm.Success.Printfln("Upgrade completed")
	}

	return nil
}

func initActionConfig(namespace string, debug bool) (*action.Configuration, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	settings.Debug = debug

	actionConfig := new(action.Configuration)
	logFn := func(format string, v ...interface{}) {
		if debug {
			pterm.Debug.Printf(format, v...)
		}
	}

	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), logFn); err != nil {
		if debug {
			pterm.Debug.Printfln("Action config initialization failed: %v", err)
		}
		return nil, fmt.Errorf("action config initialization failed: %w", err)
	}

	return actionConfig, nil
}

func verifyReleaseExists(actionConfig *action.Configuration, releaseName, namespace string, debug bool) error {
	if debug {
		pterm.Debug.Printfln("Checking if release %s exists in namespace %s", releaseName, namespace)
	}

	listAction := action.NewList(actionConfig)
	listAction.AllNamespaces = false
	listAction.All = true
	listAction.SetStateMask()

	releases, err := listAction.Run()
	if err != nil {
		if debug {
			pterm.Debug.Printfln("Failed to list releases: %v", err)
		}
		return fmt.Errorf("failed to list releases: %w", err)
	}

	for _, r := range releases {
		if r.Name == releaseName && r.Namespace == namespace {
			if debug {
				pterm.Debug.Printfln("Release found: %s (version %d)", r.Name, r.Version)
			}
			return nil
		}
	}

	if debug {
		pterm.Debug.Printfln("Release %s not found in namespace %s", releaseName, namespace)
	}
	return fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
}

func loadAndMergeValuesWithSets(valuesFiles, setValues, setLiteralValues []string, debug bool) (map[string]interface{}, error) {
	if debug {
		pterm.Debug.Printfln("Loading and merging values from %d files", len(valuesFiles))
		for i, file := range valuesFiles {
			pterm.Debug.Printfln("  %d: %s", i+1, file)
		}
	}

	valueOpts := &values.Options{
		ValueFiles:    valuesFiles,
		StringValues:  setValues,
		LiteralValues: setLiteralValues,
	}

	vals, err := valueOpts.MergeValues(getter.All(cli.New()))
	if err != nil {
		if debug {
			pterm.Debug.Printfln("Values merging failed: %v", err)
		}
		return nil, fmt.Errorf("values merging failed: %w", err)
	}

	if debug && len(setValues) > 0 {
		pterm.Debug.Printfln("Applied --set values:")
		for _, v := range setValues {
			pterm.Debug.Printfln("  - %s", v)
		}
	}

	if debug && len(setLiteralValues) > 0 {
		pterm.Debug.Printfln("Applied --set-literal values:")
		for _, v := range setLiteralValues {
			pterm.Debug.Printfln("  - %s", v)
		}
	}

	return vals, nil
}

func getPods(namespace, releaseName string) ([]corev1.Pod, error) {
	clientset, err := getKubeClient()
	if err != nil {
		return nil, err
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func printPodDetails(pod corev1.Pod, debug bool) {
	if !debug {
		return
	}

	pterm.Debug.Printfln("Pod: %s", pod.Name)
	pterm.Debug.Printfln("  Status: %s", pod.Status.Phase)
	pterm.Debug.Printfln("  Node: %s", pod.Spec.NodeName)
	pterm.Debug.Printfln("  IP: %s", pod.Status.PodIP)

	for _, cond := range pod.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			pterm.Debug.Printfln("  Condition %s: %s (Reason: %s)", cond.Type, cond.Status, cond.Reason)
		}
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			pterm.Debug.Printfln("  Container %s: NOT READY", cs.Name)
			if cs.State.Waiting != nil {
				pterm.Debug.Printfln("    Waiting Reason: %s", cs.State.Waiting.Reason)
				pterm.Debug.Printfln("    Waiting Message: %s", cs.State.Waiting.Message)
			}
		}
	}
}

func monitorResourcesUpgrade(rel *release.Release, namespace string, timeout time.Duration, debug bool) error {
	if debug {
		pterm.Debug.Println("Starting resource monitoring...")
	}

	clientset, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get kube client: %w", err)
	}

	// Parse the release manifest to get all resources
	manifests := strings.Split(rel.Manifest, "\n---\n")
	resources := make([]map[string]interface{}, 0)

	for _, manifest := range manifests {
		if strings.TrimSpace(manifest) == "" {
			continue
		}

		// Parse manifest with proper type handling
		var raw interface{}
		if err := yaml.Unmarshal([]byte(manifest), &raw); err != nil {
			if debug {
				pterm.Debug.Printfln("Failed to parse manifest: %v", err)
			}
			continue
		}

		// Convert map[interface{}]interface{} to map[string]interface{}
		converted := convertMapInterfaceToMapString(raw)
		if obj, ok := converted.(map[string]interface{}); ok {
			resources = append(resources, obj)
		}
	}

	if debug {
		pterm.Debug.Printfln("Monitoring %d resources in namespace %s", len(resources), namespace)
	}

	// Wait for all resources to become ready
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for resources to become ready")

		case <-ticker.C:
			allReady := true
			failedResources := []string{}

			for _, resource := range resources {
				ready, err := isResourceReady(clientset, resource, namespace, debug)
				if err != nil {
					if debug {
						pterm.Debug.Printfln("Error checking resource: %v", err)
					}
					failedResources = append(failedResources, fmt.Sprintf("%s: %v", getResourceName(resource), err))
					allReady = false
				} else if !ready {
					allReady = false
					if debug {
						pterm.Debug.Printfln("Resource not ready: %s", getResourceName(resource))
					}
				}
			}

			if allReady {
				if debug {
					pterm.Debug.Println("All resources are ready")
				}
				return nil
			}

			if len(failedResources) > 0 {
				return fmt.Errorf("resources failed: %s", strings.Join(failedResources, ", "))
			}

			if debug {
				pterm.Debug.Println("Still waiting for resources to become ready...")
			}
		}
	}
}

// convertMapInterfaceToMapString recursively converts map[interface{}]interface{} to map[string]interface{}
func convertMapInterfaceToMapString(input interface{}) interface{} {
	switch x := input.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, v := range x {
			m[fmt.Sprintf("%v", k)] = convertMapInterfaceToMapString(v)
		}
		return m
	case []interface{}:
		for i, v := range x {
			x[i] = convertMapInterfaceToMapString(v)
		}
		return x
	default:
		return input
	}
}
