package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// UninstallOptions contains configuration for the uninstall operation
type UninstallOptions struct {
	ReleaseName  string
	Namespace    string
	Force        bool
	Timeout      time.Duration
	DisableHooks bool
	Cascade      string // "background", "foreground", or "orphan"
}

// HelmUninstall performs a complete uninstallation of a Helm release
func HelmUninstall(opts UninstallOptions) error {
	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("Uninstalling release %s...", opts.ReleaseName))
	defer spinner.Stop()

	// Initialize Helm configuration
	actionConfig := new(action.Configuration)
	if err := initializeActionConfig(actionConfig, opts.Namespace); err != nil {
		return fmt.Errorf("failed to initialize Helm: %w", err)
	}

	// Verify release exists
	if err := verifyReleaseExists(actionConfig, opts.ReleaseName, opts.Namespace); err != nil {
		if !opts.Force {
			return err
		}
		pterm.Warning.Printfln("Release verification failed, continuing with force deletion")
	}

	// Perform Helm uninstall
	resp, err := performUninstall(actionConfig, opts)
	if err != nil {
		if !opts.Force {
			return fmt.Errorf("helm uninstall failed: %w", err)
		}
		pterm.Warning.Printfln("Helm uninstall failed, attempting force cleanup")
	}

	// Verify and cleanup remaining resources
	if err := verifyAndCleanupResources(opts, resp); err != nil {
		return fmt.Errorf("resource cleanup failed: %w", err)
	}

	pterm.Success.Printfln("Successfully uninstalled release %s from namespace %s", opts.ReleaseName, opts.Namespace)
	return nil
}

func initializeActionConfig(actionConfig *action.Configuration, namespace string) error {
	// Set up Helm environment
	if settings.KubeConfig == "" {
		settings.KubeConfig = filepath.Join(homeDir(), ".kube", "config")
	}
	if settings.KubeContext == "" {
		settings.KubeContext = os.Getenv("KUBECONTEXT")
	}

	return actionConfig.Init(
		settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			pterm.Debug.Printfln(format, v...)
		},
	)
}

func verifyReleaseExist(actionConfig *action.Configuration, releaseName string) error {
	statusAction := action.NewStatus(actionConfig)
	_, err := statusAction.Run(releaseName)
	if err != nil {
		return fmt.Errorf("release '%s' not found or inaccessible: %w", releaseName, err)
	}
	return nil
}

func performUninstall(actionConfig *action.Configuration, opts UninstallOptions) (*release.UninstallReleaseResponse, error) {
	client := action.NewUninstall(actionConfig)
	client.Wait = true
	client.Timeout = opts.Timeout
	client.DisableHooks = opts.DisableHooks
	client.KeepHistory = false
	client.Description = "smurf selm uninstall"

	// Convert cascade option to the string value Helm expects
	// Note: Helm expects the actual string values "background", "foreground", "orphan"
	switch opts.Cascade {
	case "foreground":
		client.DeletionPropagation = "foreground" // Use string literal instead of metav1 constant
	case "orphan":
		client.DeletionPropagation = "orphan" // Use string literal instead of metav1 constant
	default: // "background" or empty
		client.DeletionPropagation = "background" // Use string literal instead of metav1 constant
	}

	return client.Run(opts.ReleaseName)
}

func verifyAndCleanupResources(opts UninstallOptions, resp *release.UninstallReleaseResponse) error {
	clientset, err := getKubeClient()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	remainingResources, err := checkRemainingResources(clientset, opts.Namespace, opts.ReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check remaining resources: %w", err)
	}

	if len(remainingResources) > 0 {
		pterm.Warning.Printfln("%d resources still exist after uninstall", len(remainingResources))

		if opts.Force {
			if err := forceDeleteResources(clientset, remainingResources, opts.Cascade); err != nil {
				return fmt.Errorf("force deletion failed: %w", err)
			}
		} else {
			pterm.Warning.Printfln("Use --force flag to attempt removal of remaining resources")
		}
	}

	return nil
}
func checkRemainingResources(clientset *kubernetes.Clientset, namespace, releaseName string) ([]Resource, error) {
	var remaining []Resource

	// Check deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	for _, d := range deployments.Items {
		remaining = append(remaining, Resource{
			Kind:      "Deployment",
			Name:      d.Name,
			Namespace: d.Namespace,
		})
	}

	// Check services
	services, err := clientset.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	for _, s := range services.Items {
		remaining = append(remaining, Resource{
			Kind:      "Service",
			Name:      s.Name,
			Namespace: s.Namespace,
		})
	}

	// Add checks for other resource types (ConfigMaps, Secrets, etc.) similarly

	return remaining, nil
}

func forceDeleteResources(clientset *kubernetes.Clientset, resources []Resource, cascade string) error {
	// Convert cascade string to proper DeletionPropagation type
	var propagationPolicy metav1.DeletionPropagation
	switch cascade {
	case "foreground":
		propagationPolicy = metav1.DeletePropagationForeground
	case "orphan":
		propagationPolicy = metav1.DeletePropagationOrphan
	default:
		propagationPolicy = metav1.DeletePropagationBackground
	}

	for _, r := range resources {
		switch r.Kind {
		case "Deployment":
			if err := forceDeleteDeployment(clientset, r, propagationPolicy); err != nil {
				return err
			}
		case "Service":
			if err := forceDeleteService(clientset, r, propagationPolicy); err != nil {
				return err
			}
		// Add cases for other resource types
		default:
			pterm.Warning.Printfln("Skipping force deletion of unsupported resource type: %s/%s", r.Kind, r.Name)
		}
	}
	return nil
}

func forceDeleteDeployment(clientset *kubernetes.Clientset, r Resource, propagationPolicy metav1.DeletionPropagation) error {
	// Remove finalizers first
	dep, err := clientset.AppsV1().Deployments(r.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get deployment %s: %w", r.Name, err)
	}

	if len(dep.Finalizers) > 0 {
		pterm.Debug.Printfln("Removing finalizers from deployment %s", r.Name)
		dep.Finalizers = []string{}
		if _, err := clientset.AppsV1().Deployments(r.Namespace).Update(
			context.Background(), dep, metav1.UpdateOptions{},
		); err != nil {
			return fmt.Errorf("failed to remove finalizers: %w", err)
		}
	}

	// Delete with specified propagation policy
	if err := clientset.AppsV1().Deployments(r.Namespace).Delete(
		context.Background(), r.Name, metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		},
	); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete deployment %s: %w", r.Name, err)
	}

	pterm.Success.Printfln("Force deleted deployment %s", r.Name)
	return nil
}

func forceDeleteService(clientset *kubernetes.Clientset, r Resource, propagationPolicy metav1.DeletionPropagation) error {
	// Services typically don't have finalizers, but we'll check
	svc, err := clientset.CoreV1().Services(r.Namespace).Get(context.Background(), r.Name, metav1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get service %s: %w", r.Name, err)
	}

	if len(svc.Finalizers) > 0 {
		pterm.Debug.Printfln("Removing finalizers from service %s", r.Name)
		svc.Finalizers = []string{}
		if _, err := clientset.CoreV1().Services(r.Namespace).Update(
			context.Background(), svc, metav1.UpdateOptions{},
		); err != nil {
			return fmt.Errorf("failed to remove finalizers: %w", err)
		}
	}

	if err := clientset.CoreV1().Services(r.Namespace).Delete(
		context.Background(), r.Name, metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		},
	); err != nil && !isNotFound(err) {
		return fmt.Errorf("failed to delete service %s: %w", r.Name, err)
	}

	pterm.Success.Printfln("Force deleted service %s", r.Name)
	return nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
