package helm

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func pod(name string, phase corev1.PodPhase, waitingReason string) corev1.Pod {
	p := corev1.Pod{}
	p.Name = name
	p.Status.Phase = phase
	if waitingReason != "" {
		p.Status.ContainerStatuses = []corev1.ContainerStatus{{
			State: corev1.ContainerState{
				Waiting: &corev1.ContainerStateWaiting{Reason: waitingReason, Message: "detail"},
			},
		}}
	}
	return p
}

func TestCollectPodFailures_PicksUnhealthyPods(t *testing.T) {
	pods := []corev1.Pod{
		pod("healthy", corev1.PodRunning, ""),
		pod("pulling", corev1.PodPending, "ImagePullBackOff"),
		pod("crashing", corev1.PodRunning, "CrashLoopBackOff"),
		pod("finished", corev1.PodSucceeded, ""),
	}

	got := collectPodFailures(context.Background(), pods, nil)

	names := make([]string, 0, len(got))
	for _, f := range got {
		names = append(names, f.Name)
	}
	joined := strings.Join(names, ",")

	for _, want := range []string{"pulling", "crashing"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q to be reported, got %v", want, names)
		}
	}
	// A completed job pod is not a failure, and reporting healthy pods would
	// bury the ones that matter.
	for _, unwanted := range []string{"healthy", "finished"} {
		if strings.Contains(joined, unwanted) {
			t.Errorf("%q should not be reported as a failure, got %v", unwanted, names)
		}
	}
}

func TestCollectPodFailures_UsesTheReasonLookup(t *testing.T) {
	pods := []corev1.Pod{pod("api", corev1.PodPending, "ImagePullBackOff")}

	got := collectPodFailures(context.Background(), pods, func(context.Context, *corev1.Pod) string {
		return "manifest unknown"
	})

	if len(got) != 1 {
		t.Fatalf("expected one failure, got %d", len(got))
	}
	if got[0].Reason != "manifest unknown" {
		t.Errorf("Reason = %q, want the looked-up reason", got[0].Reason)
	}
}

func TestCollectPodFailures_NoPods(t *testing.T) {
	if got := collectPodFailures(context.Background(), nil, nil); len(got) != 0 {
		t.Errorf("expected no failures, got %v", got)
	}
}

// The pod reason is the answer someone opened the failed job to find, so it has
// to appear before Helm's own message, which is often just "timed out waiting
// for the condition".
func TestRenderFailureSummary_LeadsWithPodDetail(t *testing.T) {
	failures := []PodFailure{
		{Name: "api-7d9f", Status: "ImagePullBackOff", Reason: "manifest unknown"},
	}

	got := renderFailureSummary("Helm upgrade", "api", "prod", failures,
		errors.New("timed out waiting for the condition"))

	if !strings.Contains(got, "api-7d9f") || !strings.Contains(got, "manifest unknown") {
		t.Errorf("summary should name the pod and its reason:\n%s", got)
	}
	if !strings.Contains(got, "`api`") || !strings.Contains(got, "`prod`") {
		t.Errorf("summary should name the release and namespace:\n%s", got)
	}

	podIdx := strings.Index(got, "api-7d9f")
	helmIdx := strings.Index(got, "timed out waiting")
	if podIdx == -1 || helmIdx == -1 || podIdx > helmIdx {
		t.Errorf("pod detail should come before the helm error:\n%s", got)
	}
}

// A release can fail without any unhealthy workload, and saying so is more
// useful than an empty table that looks like a bug in the report.
func TestRenderFailureSummary_NoFailingPodsSaysSo(t *testing.T) {
	got := renderFailureSummary("Helm upgrade", "api", "prod", nil, errors.New("chart is invalid"))

	if !strings.Contains(got, "No unhealthy pods") {
		t.Errorf("summary should explain that no workload was unhealthy:\n%s", got)
	}
	if !strings.Contains(got, "chart is invalid") {
		t.Errorf("the helm error should still be shown:\n%s", got)
	}
}

func TestRenderFailureSummary_HandlesNilError(t *testing.T) {
	got := renderFailureSummary("Helm upgrade", "api", "prod",
		[]PodFailure{{Name: "api", Status: "Pending"}}, nil)

	if strings.Contains(got, "Helm error") {
		t.Errorf("no error section should appear when there is no error:\n%s", got)
	}
	if !strings.Contains(got, "api") {
		t.Errorf("pod detail should still render:\n%s", got)
	}
}

// A Kubernetes message containing a pipe would otherwise split the row and
// shift every later column.
func TestRenderFailureSummary_SurvivesPipesInMessages(t *testing.T) {
	failures := []PodFailure{
		{Name: "api", Status: "Error", Reason: "exec failed: sh -c 'a | b'"},
	}

	got := renderFailureSummary("Helm upgrade", "api", "prod", failures, nil)

	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "exec failed") {
			if strings.Count(line, "|")-strings.Count(line, "\\|") != 4 {
				t.Errorf("row should keep four unescaped delimiters: %q", line)
			}
		}
	}
}

// A large rollout can fail every pod for the same reason; fifty identical
// annotations bury the run rather than explaining it.
func TestAnnotateFailures_CapsOutput(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")

	many := make([]PodFailure, 25)
	for i := range many {
		many[i] = PodFailure{Name: "pod", Status: "CrashLoopBackOff"}
	}
	// Outside Actions this is a no-op; the assertion is that the cap logic runs
	// without panicking on a list longer than the cap.
	annotateFailures("Helm upgrade", many)
	annotateFailures("Helm upgrade", nil)
}

// Reporting is layered on a command that already failed, so a missing cluster
// or absent Actions environment must not change anything.
func TestReportFailureToCI_SilentOutsideActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_STEP_SUMMARY", "")

	ReportFailureToCI("prod", "api", "Helm upgrade", errors.New("boom"))
}

// A Helm upgrade can fail before any workload exists: an unreachable cluster,
// a chart that will not load, values that will not parse. End-to-end testing
// showed those produced no CI output at all, which is the opposite of what is
// wanted, since they are the runs hardest to diagnose from the log.
func TestReportFailureToCI_WritesSummaryWithNoPods(t *testing.T) {
	resetReportedForTest()
	path := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_STEP_SUMMARY", path)

	ReportFailureToCI("prod", "api", "Helm upgrade",
		errors.New("failed to list releases: Kubernetes cluster unreachable"))

	data, err := os.ReadFile(path) // #nosec G304 -- this test's own t.TempDir file
	if err != nil {
		t.Fatalf("no summary written: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "Helm upgrade failed") {
		t.Errorf("summary should report the failure:\n%s", got)
	}
	if !strings.Contains(got, "cluster unreachable") {
		t.Errorf("summary should carry the underlying error:\n%s", got)
	}
}

// One failure should produce one report. The command layer and HelmUpgrade
// both report so that either can be the first to see a failure, which means
// paths passing through both would otherwise publish twice.
func TestReportFailureToCI_ReportsOnlyOnce(t *testing.T) {
	resetReportedForTest()
	path := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_STEP_SUMMARY", path)

	ReportFailureToCI("prod", "api", "Helm upgrade", errors.New("first failure"))
	ReportFailureToCI("prod", "api", "Helm upgrade", errors.New("second failure"))

	data, _ := os.ReadFile(path) // #nosec G304 -- this test's own t.TempDir file
	got := string(data)

	if n := strings.Count(got, "Helm upgrade failed"); n != 1 {
		t.Errorf("expected exactly one report, found %d:\n%s", n, got)
	}
	if strings.Contains(got, "second failure") {
		t.Error("a later failure is usually a consequence of the first; the first is the one to lead with")
	}
}

func TestFirstLine(t *testing.T) {
	if got := firstLine("one\ntwo\nthree"); got != "one" {
		t.Errorf("firstLine = %q, want the first line only", got)
	}
	long := strings.Repeat("x", 500)
	got := firstLine(long)
	if len(got) > 310 {
		t.Errorf("firstLine length = %d, want it truncated", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncation should be visible")
	}
}
