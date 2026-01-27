package helm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HelmReleaseExists checks if a Helm release with the given name exists in the specified namespace.
// It initializes Helm's action configuration, runs the Helm list command, and returns true if a release
// with the given name is found in the specified namespace.
// Helper function to check if release exists
func HelmReleaseExists(releaseName, namespace string, debug, useAI bool) (bool, error) {
	if debug {
		pterm.Printf("=== CHECKING RELEASE EXISTENCE ===\n")
		pterm.Printf("Release: %s, Namespace: %s\n", releaseName, namespace)
	}

	settings := cli.New()
	settings.SetNamespace(namespace)

	actionConfig := new(action.Configuration)
	err := actionConfig.Init(
		settings.RESTClientGetter(),
		namespace,
		os.Getenv("HELM_DRIVER"),
		func(format string, v ...interface{}) {
			if debug {
				pterm.Printf("HELM: %s\n", strings.TrimSpace(fmt.Sprintf(format, v...)))
			}
		},
	)
	if err != nil {
		if debug {
			pterm.Printf("Failed to initialize Helm config: %v\n", err)
		}
		return false, fmt.Errorf("failed to initialize helm config: %w", err)
	}

	listAction := action.NewList(actionConfig)
	listAction.AllNamespaces = false
	listAction.All = true // Important: list all statuses
	listAction.SetStateMask()

	releases, err := listAction.Run()
	if err != nil {
		if debug {
			pterm.Printf("Failed to list releases: %v\n", err)
		}
		return false, fmt.Errorf("failed to list releases: %w", err)
	}

	if debug {
		pterm.Printf("Found %d total releases in namespace %s\n", len(releases), namespace)
		for i, r := range releases {
			pterm.Printf("  %d. %s (Namespace: %s, Status: %s, Version: %d)\n",
				i+1, r.Name, r.Namespace, r.Info.Status, r.Version)
		}
	}

	for _, r := range releases {
		if r.Name == releaseName && r.Namespace == namespace {
			if debug {
				pterm.Printf("✅ Found matching release: %s in namespace %s\n", releaseName, namespace)
			}
			return true, nil
		}
	}

	if debug {
		pterm.Printf("❌ Release %s not found in namespace %s\n", releaseName, namespace)
		pterm.Printf("Checking if resources exist without Helm metadata...\n")
	}

	// Fallback: check if any resources exist with the release name
	clientset, err := getKubeClient()
	if err != nil {
		if debug {
			pterm.Printf("Failed to get kube client: %v\n", err)
		}
		return false, nil
	}

	// Check multiple label selectors
	labelSelectors := []string{
		fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
		fmt.Sprintf("release=%s", releaseName), // Older charts use this
	}

	for _, selector := range labelSelectors {
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err == nil && len(deployments.Items) > 0 {
			if debug {
				pterm.Printf("✅ Found %d deployments with selector: %s\n", len(deployments.Items), selector)
				for _, dep := range deployments.Items {
					pterm.Printf("  - %s (labels: %v)\n", dep.Name, dep.Labels)
				}
			}
			return true, nil
		}
	}

	if debug {
		pterm.Printf("❌ No resources found for release %s in namespace %s\n", releaseName, namespace)
	}

	return false, nil
}
