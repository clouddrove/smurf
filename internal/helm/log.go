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

var appKube string = "app.kubernetes.io/instance="

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
		LabelSelector: appKube + release,
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
			podColor := getResourceColor("pod")

			// Determine pod status from container statuses first, fall back to phase
			status := string(pod.Status.Phase)
			var messages []string

			// Check container statuses for actual state
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					// Use Waiting reason as the main status (e.g., CrashLoopBackOff, ImagePullBackOff)
					status = cs.State.Waiting.Reason
					// Store the message for Info column
					if cs.State.Waiting.Message != "" {
						messages = append(messages, cs.State.Waiting.Message)
					}
				} else if cs.State.Terminated != nil {
					status = cs.State.Terminated.Reason
					// Store the message for Info column
					if cs.State.Terminated.Message != "" {
						messages = append(messages, cs.State.Terminated.Message)
					}
				}

			}

			statusColor := getPodStatusColor(status)

			if len(messages) == 0 {
				// no messages - just show status
				fmt.Fprintf(w, "  └─ %sPod/ %s%s\t%s%s%s\t\n",
					podColor, pod.Name, ColorReset,
					statusColor, status, ColorReset)
			} else {
				// Combine all messages into one string with line breaks
				fullMessage := strings.Join(messages, "\n")
				wrappedLines := wrapText(fullMessage, 70)

				// Print first line with pod info and first message line in Info column
				fmt.Fprintf(w, "  └─ %sPod/ %s%s\t%s%s%s\t%s%s%s\n",
					podColor, pod.Name, ColorReset,
					statusColor, status, ColorReset,
					ColorRed, wrappedLines[0], ColorReset)

				// Print remaining message lines with manual spacing to align with Info column
				for i := 1; i < len(wrappedLines); i++ {
					fmt.Fprintf(w, "                                                                   %s%s%s\n", ColorRed, wrappedLines[i], ColorReset)
				}
			}
		}
	}

	// --- Services ---
	services, _ := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: appKube + release,
	})
	for _, svc := range services.Items {
		serviceColor := getResourceColor("service")
		fmt.Fprintf(w, "%sservice/ %s%s\t\t\n", serviceColor, svc.Name, ColorReset)
	}

	// --- ConfigMaps ---
	configMaps, _ := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: appKube + release,
	})
	for _, cm := range configMaps.Items {
		cmColor := getResourceColor("configmap")
		fmt.Fprintf(w, "%sconfigmap/ %s%s\t\t\n", cmColor, cm.Name, ColorReset)
	}

	// --- Ingresses ---
	ingresses, _ := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: appKube + release,
	})
	for _, ing := range ingresses.Items {
		ingressColor := getResourceColor("ingress")
		fmt.Fprintf(w, "%singress/ %s%s\t\t\n", ingressColor, ing.Name, ColorReset)
	}

	w.Flush()
}
func printErrorSummary(stage, releaseName, namespace, chartName string, err error) {
	fmt.Println("")
	fmt.Println(pterm.Red("INSTALLATION FAILED"))
	fmt.Println("-------------------")
	fmt.Println("Stage :        ", stage)
	fmt.Println("Release Name : ", releaseName)
	fmt.Println("Namespace :    ", namespace)
	fmt.Println("Chart :        ", chartName)
	fmt.Println(pterm.Red("Error :         ", err))
}
