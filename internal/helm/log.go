package helm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/pterm/pterm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ANSI color codes
const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorGray    = "\033[90m"

	ColorBold   = "\033[1m"
	ColorItalic = "\033[3m"
)

// getPodStatusColor returns color based on pod status
func getPodStatusColor(status string) string {
	switch status {
	case "Running", "Succeeded":
		return ColorGreen
	case "Pending":
		return ColorYellow
	case "Failed", "Error", "CrashLoopBackOff":
		return ColorRed
	case "Unknown":
		return ColorGray
	default:
		return ColorWhite
	}
}

// getResourceColor returns color for resource types
func getResourceColor(resourceType string) string {
	switch resourceType {
	case "deployment":
		return ColorCyan
	case "service":
		return ColorBlue
	case "configmap":
		return ColorMagenta
	case "ingress":
		return ColorYellow
	case "pod":
		return ColorWhite
	default:
		return ColorWhite
	}
}

// wrapText splits a long string into chunks with max width
func wrapText(msg string, width int) []string {
	var lines []string
	for len(msg) > width {
		cut := strings.LastIndex(msg[:width], " ")
		if cut == -1 {
			cut = width
		}
		lines = append(lines, msg[:cut])
		msg = strings.TrimSpace(msg[cut:])
	}
	if len(msg) > 0 {
		lines = append(lines, msg)
	}
	return lines
}

// printReleaseResources prints all resources of a Helm release with statuses & errors
func printReleaseResources(namespace, release string) {
	ctx := context.Background()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	// colorful table header
	headerColor := ColorBold + ColorCyan
	fmt.Fprintln(w, headerColor+"Resource\tStatus\tInfo"+ColorReset)
	fmt.Fprintln(w, headerColor+"--------\t------\t----"+ColorReset)

	clientset, _ := getKubeClient()

	// --- Deployments ---
	deploys, _ := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance=" + release,
	})
	for _, dep := range deploys.Items {
		deployColor := getResourceColor("deployment")
		fmt.Fprintf(w, "%sdeployment/ %s%s\t\t\n", deployColor, dep.Name, ColorReset)

		// get pods of this deployment
		labelSelector := ""
		if dep.Spec.Selector != nil {
			labelSelector = metav1.FormatLabelSelector(dep.Spec.Selector)
		}
		pods, _ := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})

		for _, pod := range pods.Items {
			status := string(pod.Status.Phase)
			statusColor := getPodStatusColor(status)
			podColor := getResourceColor("pod")

			// collect all errors
			var errors []string
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					errors = append(errors, fmt.Sprintf("%s: %s", cs.State.Waiting.Reason, cs.State.Waiting.Message))
				}
				if cs.State.Terminated != nil {
					errors = append(errors, fmt.Sprintf("%s: %s", cs.State.Terminated.Reason, cs.State.Terminated.Message))
				}
			}

			if len(errors) == 0 {
				// no error - green status
				fmt.Fprintf(w, "  └─ %sPod/ %s%s\t%s%s%s\t\n",
					podColor, pod.Name, ColorReset,
					statusColor, status, ColorReset)
			} else {
				// first error goes into Info column
				wrapped := wrapText(errors[0], 70)

				// print first line with error (red for errors)
				fmt.Fprintf(w, "  └─ %sPod/ %s%s\t%s%s%s\t%s%s%s\n",
					podColor, pod.Name, ColorReset,
					statusColor, status, ColorReset,
					ColorRed, wrapped[0], ColorReset)

				// continuation lines -> same position as first line (aligned under Info column)
				if len(wrapped) > 1 {
					for _, l := range wrapped[1:] {
						fmt.Fprintf(w, "\t\t%s%s%s\n", ColorRed, l, ColorReset)
					}
				}

				// print additional errors
				for _, e := range errors[1:] {
					for _, l := range wrapText(e, 70) {
						fmt.Fprintf(w, "\t\t%s%s%s\n", ColorRed, l, ColorReset)
					}
				}
			}
		}
	}

	// --- Services ---
	services, _ := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance=" + release,
	})
	for _, svc := range services.Items {
		serviceColor := getResourceColor("service")
		fmt.Fprintf(w, "%sservice/ %s%s\t\t\n", serviceColor, svc.Name, ColorReset)
	}

	// --- ConfigMaps ---
	configMaps, _ := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance=" + release,
	})
	for _, cm := range configMaps.Items {
		cmColor := getResourceColor("configmap")
		fmt.Fprintf(w, "%sconfigmap/ %s%s\t\t\n", cmColor, cm.Name, ColorReset)
	}

	// --- Ingresses ---
	ingresses, _ := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance=" + release,
	})
	for _, ing := range ingresses.Items {
		ingressColor := getResourceColor("ingress")
		fmt.Fprintf(w, "%singress/ %s%s\t\t\n", ingressColor, ing.Name, ColorReset)
	}

	w.Flush()
}

func errorLock(stage, releaseName, namespace, chartName string, err error) {
	fmt.Println("")
	fmt.Println(pterm.Red("INSTALLATION FAILED"))
	fmt.Println("-------------------")
	fmt.Println("Stage :        ", stage)
	fmt.Println("Release Name : ", releaseName)
	fmt.Println("Namespace :    ", namespace)
	fmt.Println("Chart :        ", chartName)
	fmt.Println(pterm.Red("Error :         ", err))
}
