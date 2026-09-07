package helm

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clouddrove/smurf/internal/ci"
)

// Reporting a Helm failure to CI is deliberately separate from the pterm
// output the existing diagnostics produce. Those are written for a terminal,
// where colour and layout carry the meaning; in GitHub Actions the same text
// is a flat wall of log with escape codes, and the line that matters is
// hundreds of lines up.
//
// Nothing here changes what a developer sees locally. Outside Actions every
// call is a no-op.

// PodFailure is the part of a pod's state worth surfacing when a release fails.
type PodFailure struct {
	Name   string
	Status string
	Reason string
}

// collectPodFailures returns the pods of a release that are not healthy.
//
// The pod list is taken as an argument rather than fetched here so the
// selection logic can be tested without a cluster, which is the part that
// decides whether a failure is reported at all.
func collectPodFailures(ctx context.Context, pods []corev1.Pod, reason func(context.Context, *corev1.Pod) string) []PodFailure {
	var failures []PodFailure
	for i := range pods {
		pod := pods[i]
		if !isPodInFailureState(&pod) && pod.Status.Phase == corev1.PodRunning {
			continue
		}
		if pod.Status.Phase == corev1.PodSucceeded {
			continue
		}
		f := PodFailure{
			Name:   pod.Name,
			Status: getKubectlLikeStatus(pod),
		}
		if reason != nil {
			f.Reason = reason(ctx, &pod)
		}
		failures = append(failures, f)
	}
	return failures
}

// renderFailureSummary builds the markdown shown at the top of the Actions run.
//
// The failing pods come first and the Helm error last: Helm's own message is
// often "timed out waiting for the condition", which says nothing, while the
// pod reason is the answer someone opened the job to find.
func renderFailureSummary(stage, releaseName, namespace string, failures []PodFailure, helmErr error) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## ❌ %s failed\n\n", stage)
	fmt.Fprintf(&b, "**Release:** `%s`  **Namespace:** `%s`\n\n", releaseName, namespace)

	if len(failures) > 0 {
		rows := make([][]string, 0, len(failures))
		for _, f := range failures {
			rows = append(rows, []string{"`" + f.Name + "`", f.Status, f.Reason})
		}
		b.WriteString(ci.Table([]string{"Pod", "Status", "Reason"}, rows))
		b.WriteString("\n")
	} else {
		b.WriteString("No unhealthy pods were found for this release, so the failure is likely in the chart or the release itself rather than in a workload.\n\n")
	}

	if helmErr != nil {
		b.WriteString("<details><summary>Helm error</summary>\n\n")
		b.WriteString(ci.CodeBlock("text", helmErr.Error()))
		b.WriteString("\n</details>\n")
	}
	return b.String()
}

// annotateFailures attaches one annotation per failing pod so the reason shows
// in the Actions UI and on the pull request, not only in the log.
//
// A cap is applied because a large rollout can fail every pod for the same
// reason, and fifty identical annotations bury the run rather than explain it.
func annotateFailures(stage string, failures []PodFailure) {
	const maxAnnotations = 10

	if len(failures) == 0 {
		return
	}
	for i, f := range failures {
		if i == maxAnnotations {
			ci.Annotate(ci.LevelWarning, stage,
				fmt.Sprintf("%d more failing pods not annotated; see the job summary for the full list", len(failures)-maxAnnotations))
			break
		}
		msg := f.Name + ": " + f.Status
		if f.Reason != "" {
			msg += " - " + f.Reason
		}
		ci.Annotate(ci.LevelError, stage, msg)
	}
}

// ReportFailureToCI publishes a Helm failure to GitHub Actions.
//
// Failures inside this function are swallowed on purpose. Reporting is a
// diagnostic aid layered on top of a command that has already failed; being
// unable to describe the failure must not change what the command returns, and
// the pterm diagnostics have already run regardless.
func ReportFailureToCI(namespace, releaseName, stage string, helmErr error) {
	if !ci.IsGitHubActions() && !ci.SummaryEnabled() {
		return
	}

	var failures []PodFailure
	if clientset, err := getKubeClient(); err == nil {
		ctx := context.Background()
		podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
		})
		if err == nil {
			failures = collectPodFailures(ctx, podList.Items, func(c context.Context, p *corev1.Pod) string {
				return getPodFailureReason(c, clientset, p)
			})
		}
	}

	annotateFailures(stage, failures)
	ci.Summary(renderFailureSummary(stage, releaseName, namespace, failures, helmErr))
}
