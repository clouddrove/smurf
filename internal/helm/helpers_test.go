package helm

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStringFormat(t *testing.T) {
	if got := stringFormat("container", "CrashLoopBackOff"); got != "container: CrashLoopBackOff" {
		t.Errorf("stringFormat = %q", got)
	}
}

func TestMin(t *testing.T) {
	cases := []struct{ a, b, want int }{
		{1, 2, 1}, {5, 3, 3}, {4, 4, 4}, {-1, 0, -1},
	}
	for _, c := range cases {
		if got := min(c.a, c.b); got != c.want {
			t.Errorf("min(%d,%d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestWrapText(t *testing.T) {
	cases := []struct {
		name  string
		msg   string
		width int
		want  []string
	}{
		{"short fits", "hello", 10, []string{"hello"}},
		{"empty", "", 5, nil},
		{"wraps on space", "hello world foo", 8, []string{"hello", "world", "foo"}},
		{"no space hard cut", "abcdefgh", 3, []string{"abc", "def", "gh"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := wrapText(c.msg, c.width)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("wrapText(%q,%d) = %v, want %v", c.msg, c.width, got, c.want)
			}
		})
	}
}

func TestGetPodStatusColor(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{"Running", ColorGreen},
		{"Succeeded", ColorGreen},
		{"Pending", ColorYellow},
		{"CrashLoopBackOff", ColorRed},
		{"Unknown", ColorGray},
		{"SomethingElse", ColorWhite},
	}
	for _, c := range cases {
		if got := getPodStatusColor(c.status); got != c.want {
			t.Errorf("getPodStatusColor(%q) = %q, want %q", c.status, got, c.want)
		}
	}
}

func TestGetResourceColor(t *testing.T) {
	cases := []struct {
		resource string
		want     string
	}{
		{"deployment", ColorCyan},
		{"service", ColorBlue},
		{"configmap", ColorMagenta},
		{"ingress", ColorYellow},
		{"pod", ColorWhite},
		{"unknown", ColorWhite},
	}
	for _, c := range cases {
		if got := getResourceColor(c.resource); got != c.want {
			t.Errorf("getResourceColor(%q) = %q, want %q", c.resource, got, c.want)
		}
	}
}

func TestKubectlLogsCommand(t *testing.T) {
	cases := []struct {
		name      string
		namespace string
		pod       string
		container string
		previous  bool
		tail      int64
		want      string
	}{
		{"basic", "ns", "p", "", false, 0, "kubectl logs p -n ns"},
		{"with container", "ns", "p", "c", false, 0, "kubectl logs p -n ns -c c"},
		{"previous", "ns", "p", "", true, 0, "kubectl logs p -n ns --previous"},
		{"tail", "ns", "p", "", false, 100, "kubectl logs p -n ns --tail=100"},
		{"all", "ns", "p", "c", true, 50, "kubectl logs p -n ns -c c --previous --tail=50"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := kubectlLogsCommand(c.namespace, c.pod, c.container, c.previous, c.tail); got != c.want {
				t.Errorf("kubectlLogsCommand = %q, want %q", got, c.want)
			}
		})
	}
}

func TestMergeMaps(t *testing.T) {
	a := map[string]interface{}{
		"keep":   1,
		"nested": map[string]interface{}{"x": 1, "y": 2},
	}
	b := map[string]interface{}{
		"add":    2,
		"nested": map[string]interface{}{"y": 20, "z": 30},
	}
	got := mergeMaps(a, b)
	if got["keep"] != 1 || got["add"] != 2 {
		t.Errorf("top-level merge wrong: %v", got)
	}
	nested, ok := got["nested"].(map[string]interface{})
	if !ok {
		t.Fatalf("nested not a map: %v", got["nested"])
	}
	// b overrides overlapping keys, non-overlapping keys from both survive.
	if nested["x"] != 1 || nested["y"] != 20 || nested["z"] != 30 {
		t.Errorf("nested merge wrong: %v", nested)
	}
}

func TestConvertToMapStringInterface(t *testing.T) {
	in := map[interface{}]interface{}{
		"a": 1,
		"b": map[interface{}]interface{}{"c": 2},
		"d": []interface{}{map[interface{}]interface{}{"e": 3}},
	}
	got := convertToMapStringInterface(in)
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("top-level not map[string]interface{}: %T", got)
	}
	if _, ok := m["b"].(map[string]interface{}); !ok {
		t.Errorf("nested map not converted: %T", m["b"])
	}
	arr, ok := m["d"].([]interface{})
	if !ok || len(arr) != 1 {
		t.Fatalf("slice not preserved: %v", m["d"])
	}
	if _, ok := arr[0].(map[string]interface{}); !ok {
		t.Errorf("map inside slice not converted: %T", arr[0])
	}
}

func TestParseResourcesFromManifest(t *testing.T) {
	t.Run("valid multi-doc manifest", func(t *testing.T) {
		manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
---
apiVersion: v1
kind: Service
metadata:
  name: web-svc
`
		got, err := parseResourcesFromManifest(manifest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []Resource{{Kind: "Deployment", Name: "web"}, {Kind: "Service", Name: "web-svc"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("docs without kind or name are skipped", func(t *testing.T) {
		manifest := "kind: ConfigMap\n---\nfoo: bar\n---\nmetadata:\n  name: orphan\n"
		got, err := parseResourcesFromManifest(manifest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected no resources, got %v", got)
		}
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		_, err := parseResourcesFromManifest("kind: Deployment\n\tbad: indent")
		if err == nil {
			t.Fatal("expected a parse error")
		}
		if !strings.Contains(err.Error(), "failed to parse manifest") {
			t.Errorf("error %q missing context", err.Error())
		}
	})
}

func TestContainsRepo(t *testing.T) {
	repos := []string{"stable", "bitnami"}
	if !containsRepo(repos, "bitnami") {
		t.Error("expected bitnami to be found")
	}
	if containsRepo(repos, "missing") {
		t.Error("did not expect missing to be found")
	}
	if containsRepo(nil, "x") {
		t.Error("empty list should not contain anything")
	}
}

func TestPluralize(t *testing.T) {
	if got := pluralize(1, "y", "ies"); got != "y" {
		t.Errorf("pluralize(1) = %q, want y", got)
	}
	if got := pluralize(2, "y", "ies"); got != "ies" {
		t.Errorf("pluralize(2) = %q, want ies", got)
	}
	if got := pluralize(0, "", "s"); got != "s" {
		t.Errorf("pluralize(0) = %q, want s", got)
	}
}

func TestColorizeStatus(t *testing.T) {
	// pterm may disable ANSI when stdout is not a TTY, so assert the status
	// text survives rather than asserting on specific color codes.
	for _, s := range []string{"deployed", "failed", "pending", "superseded"} {
		if got := colorizeStatus(s); !strings.Contains(got, s) {
			t.Errorf("colorizeStatus(%q) = %q, dropped the status text", s, got)
		}
	}
}

func TestTruncateDescription(t *testing.T) {
	cases := []struct {
		desc   string
		maxLen int
		want   string
	}{
		{"short", 30, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"this description is definitely too long", 10, "this de..."},
	}
	for _, c := range cases {
		if got := truncateDescription(c.desc, c.maxLen); got != c.want {
			t.Errorf("truncateDescription(%q,%d) = %q, want %q", c.desc, c.maxLen, got, c.want)
		}
	}
}

func TestSafeInt(t *testing.T) {
	if got := safeInt(-5); got != 0 {
		t.Errorf("safeInt(-5) = %d, want 0", got)
	}
	if got := safeInt(7); got != 7 {
		t.Errorf("safeInt(7) = %d, want 7", got)
	}
}

func TestSafeTime(t *testing.T) {
	if got := safeTime(nil); got != "unknown" {
		t.Errorf("safeTime(nil) = %q, want unknown", got)
	}
	if got := safeTime(&release.Info{}); got != "unknown" {
		t.Errorf("safeTime(zero) = %q, want unknown", got)
	}
	fixed := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	info := &release.Info{LastDeployed: helmtime.Time{Time: fixed}}
	if got := safeTime(info); got != "2024-01-02 03:04:05" {
		t.Errorf("safeTime(set) = %q", got)
	}
}

func TestSafeStatus(t *testing.T) {
	if got := safeStatus(nil); got != "unknown" {
		t.Errorf("safeStatus(nil) = %q", got)
	}
	if got := safeStatus(&release.Info{}); got != "unknown" {
		t.Errorf("safeStatus(empty) = %q", got)
	}
	if got := safeStatus(&release.Info{Status: release.StatusDeployed}); got != "deployed" {
		t.Errorf("safeStatus(deployed) = %q", got)
	}
}

func TestSafeChartName(t *testing.T) {
	if got := safeChartName(nil); got != "unknown" {
		t.Errorf("safeChartName(nil) = %q", got)
	}
	if got := safeChartName(&chart.Chart{}); got != "unknown" {
		t.Errorf("safeChartName(no metadata) = %q", got)
	}
	c := &chart.Chart{Metadata: &chart.Metadata{Name: "web", Version: "1.2.3"}}
	if got := safeChartName(c); got != "web-1.2.3" {
		t.Errorf("safeChartName = %q, want web-1.2.3", got)
	}
}

func TestSafeAppVersion(t *testing.T) {
	if got := safeAppVersion(nil); got != "unknown" {
		t.Errorf("safeAppVersion(nil) = %q", got)
	}
	c := &chart.Chart{Metadata: &chart.Metadata{AppVersion: "9.9"}}
	if got := safeAppVersion(c); got != "9.9" {
		t.Errorf("safeAppVersion = %q, want 9.9", got)
	}
}

func TestSafeDescription(t *testing.T) {
	if got := safeDescription(nil); got != "unknown" {
		t.Errorf("safeDescription(nil) = %q", got)
	}
	if got := safeDescription(&release.Info{Description: "install complete"}); got != "install complete" {
		t.Errorf("safeDescription = %q", got)
	}
}

func TestIsNotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"os not exist", os.ErrNotExist, true},
		{"substring not found", errors.New("release: not found"), true},
		{"unrelated", errors.New("connection refused"), false},
	}
	for _, c := range cases {
		if got := isNotFound(c.err); got != c.want {
			t.Errorf("isNotFound(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestFormatTime(t *testing.T) {
	fixed := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
	if got := formatTime(helmtime.Time{Time: fixed}); got != "2023-12-25 10:30:00" {
		t.Errorf("formatTime = %q", got)
	}
}

func TestPodOrNone(t *testing.T) {
	if got := podOrNone(""); got != "<none>" {
		t.Errorf("podOrNone(empty) = %q, want <none>", got)
	}
	if got := podOrNone("value"); got != "value" {
		t.Errorf("podOrNone(value) = %q", got)
	}
}

func TestGetExternalIP(t *testing.T) {
	t.Run("loadbalancer with IP", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer}}
		svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: "1.2.3.4"}}
		if got := getExternalIP(svc); got != "1.2.3.4" {
			t.Errorf("got %q, want 1.2.3.4", got)
		}
	})
	t.Run("loadbalancer with hostname", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer}}
		svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{Hostname: "lb.example.com"}}
		if got := getExternalIP(svc); got != "lb.example.com" {
			t.Errorf("got %q, want lb.example.com", got)
		}
	})
	t.Run("loadbalancer pending", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer}}
		if got := getExternalIP(svc); got != "<pending>" {
			t.Errorf("got %q, want <pending>", got)
		}
	})
	t.Run("clusterip", func(t *testing.T) {
		svc := &corev1.Service{Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP}}
		if got := getExternalIP(svc); got != "<none>" {
			t.Errorf("got %q, want <none>", got)
		}
	})
}

func runningReadyPod() corev1.Pod {
	return corev1.Pod{
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
		},
	}
}

func TestIsPodReady(t *testing.T) {
	t.Run("succeeded", func(t *testing.T) {
		p := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}
		if !isPodReady(p) {
			t.Error("succeeded pod should be ready")
		}
	})
	t.Run("running and ready", func(t *testing.T) {
		if !isPodReady(runningReadyPod()) {
			t.Error("running+ready pod should be ready")
		}
	})
	t.Run("running not ready", func(t *testing.T) {
		p := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}
		if isPodReady(p) {
			t.Error("running pod without ready condition should not be ready")
		}
	})
	t.Run("pending", func(t *testing.T) {
		p := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}
		if isPodReady(p) {
			t.Error("pending pod should not be ready")
		}
	})
}

func TestIsPodReadyInstall(t *testing.T) {
	ready := &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}}}
	notReady := &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}}}}
	none := &corev1.Pod{}
	if !isPodReadyInstall(ready) {
		t.Error("expected ready")
	}
	if isPodReadyInstall(notReady) {
		t.Error("expected not ready")
	}
	if isPodReadyInstall(none) {
		t.Error("expected not ready when no condition present")
	}
}

func TestIsPodFromRelease(t *testing.T) {
	cases := []struct {
		name    string
		pod     corev1.Pod
		release string
		want    bool
	}{
		{
			name:    "instance label",
			pod:     corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "x", Labels: map[string]string{"app.kubernetes.io/instance": "MyApp"}}},
			release: "myapp",
			want:    true,
		},
		{
			name:    "release label",
			pod:     corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "x", Labels: map[string]string{"release": "myapp"}}},
			release: "myapp",
			want:    true,
		},
		{
			name:    "name contains release",
			pod:     corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "myapp-abc-123"}},
			release: "myapp",
			want:    true,
		},
		{
			name:    "no match",
			pod:     corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "other-xyz", Labels: map[string]string{"app": "z"}}},
			release: "myapp",
			want:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isPodFromRelease(c.pod, c.release); got != c.want {
				t.Errorf("isPodFromRelease = %v, want %v", got, c.want)
			}
		})
	}
}

func TestGetTotalRestarts(t *testing.T) {
	pod := corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
		{RestartCount: 2}, {RestartCount: 3},
	}}}
	if got := getTotalRestarts(pod); got != 5 {
		t.Errorf("getTotalRestarts = %d, want 5", got)
	}
	if got := getTotalRestarts(corev1.Pod{}); got != 0 {
		t.Errorf("getTotalRestarts(empty) = %d, want 0", got)
	}
}

func TestFindContainerStatus(t *testing.T) {
	pod := corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
		{Name: "app", Ready: true}, {Name: "sidecar"},
	}}}
	if got := findContainerStatus(pod, "app"); !got.Ready {
		t.Errorf("expected to find ready 'app', got %+v", got)
	}
	got := findContainerStatus(pod, "missing")
	if got.Name != "missing" || got.Ready {
		t.Errorf("missing container should return placeholder, got %+v", got)
	}
}

func waitingCS(reason string) corev1.ContainerStatus {
	return corev1.ContainerStatus{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}}
}

func terminatedCS(exit int32, reason string) corev1.ContainerStatus {
	return corev1.ContainerStatus{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: exit, Reason: reason}}}
}

func TestIsContainerStatusUnhealthy(t *testing.T) {
	cases := []struct {
		name string
		cs   corev1.ContainerStatus
		want bool
	}{
		{"crashloop", waitingCS("CrashLoopBackOff"), true},
		{"imagepull", waitingCS("ImagePullBackOff"), true},
		{"benign waiting", waitingCS("ContainerCreating"), false},
		{"terminated nonzero", terminatedCS(1, "Error"), true},
		{"terminated zero", terminatedCS(0, "Completed"), false},
		{"last termination nonzero", corev1.ContainerStatus{LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 2}}}, true},
		{"healthy running", corev1.ContainerStatus{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isContainerStatusUnhealthy(c.cs); got != c.want {
				t.Errorf("isContainerStatusUnhealthy = %v, want %v", got, c.want)
			}
		})
	}
}

func TestContainerNeedsPreviousLogs(t *testing.T) {
	cases := []struct {
		name string
		cs   corev1.ContainerStatus
		want bool
	}{
		{"last termination", corev1.ContainerStatus{LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}}, true},
		{"crashloop", waitingCS("CrashLoopBackOff"), true},
		{"restart count", corev1.ContainerStatus{RestartCount: 1}, true},
		{"fresh", corev1.ContainerStatus{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := containerNeedsPreviousLogs(c.cs); got != c.want {
				t.Errorf("containerNeedsPreviousLogs = %v, want %v", got, c.want)
			}
		})
	}
}

func TestContainerNeverStarted(t *testing.T) {
	cases := []struct {
		name string
		cs   corev1.ContainerStatus
		want bool
	}{
		{"imagepull", waitingCS("ImagePullBackOff"), true},
		{"podinitializing", waitingCS("PodInitializing"), true},
		{"crashloop is a start", waitingCS("CrashLoopBackOff"), false},
		{"running", corev1.ContainerStatus{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := containerNeverStarted(c.cs); got != c.want {
				t.Errorf("containerNeverStarted = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsPodInFailureState(t *testing.T) {
	cases := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{"phase failed", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}}, true},
		{"waiting imagepull", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{waitingCS("ImagePullBackOff")}}}, true},
		{"terminated error", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{terminatedCS(0, "Error")}}}, true},
		{"terminated nonzero exit", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{terminatedCS(3, "OOMKilled")}}}, true},
		{"healthy running", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isPodInFailureState(c.pod); got != c.want {
				t.Errorf("isPodInFailureState = %v, want %v", got, c.want)
			}
		})
	}
}

func TestGetPodReadyStatus(t *testing.T) {
	t.Run("all ready", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}}},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: "a", Ready: true}}},
		}
		if got := getPodReadyStatus(pod); got != "1/1 containers ready" {
			t.Errorf("got %q, want 1/1 containers ready", got)
		}
	})
	t.Run("partial with waiting reason", func(t *testing.T) {
		pod := &corev1.Pod{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "a"}, {Name: "b"}}},
			Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{
				{Name: "a", Ready: true},
				{Name: "b", Ready: false, State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}},
			}},
		}
		got := getPodReadyStatus(pod)
		if !strings.Contains(got, "1/2 containers ready") || !strings.Contains(got, "b: CrashLoopBackOff") {
			t.Errorf("got %q, want it to mention 1/2 and b: CrashLoopBackOff", got)
		}
	})
}

func TestContainerStateSummary(t *testing.T) {
	cases := []struct {
		name string
		cs   corev1.ContainerStatus
		want string
	}{
		{"waiting", corev1.ContainerStatus{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff", Message: "back-off"}}}, "Waiting (CrashLoopBackOff): back-off"},
		{"terminated", corev1.ContainerStatus{State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 1, Message: "boom"}}}, "Terminated (Error, exit 1): boom"},
		{"running ready", corev1.ContainerStatus{Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}, "Running (ready)"},
		{"running not ready", corev1.ContainerStatus{Ready: false, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}, "Running (not ready)"},
		{"empty", corev1.ContainerStatus{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := containerStateSummary(c.cs); got != c.want {
				t.Errorf("containerStateSummary = %q, want %q", got, c.want)
			}
		})
	}
}

func TestIsPodUnhealthyForUpgrade(t *testing.T) {
	cases := []struct {
		name string
		pod  *corev1.Pod
		want bool
	}{
		{"failed phase", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}}, true},
		{"waiting crashloop", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{waitingCS("CrashLoopBackOff")}}}, true},
		{"terminated nonzero", &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{terminatedCS(1, "Error")}}}, true},
		{"healthy", &corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}, ContainerStatuses: []corev1.ContainerStatus{{Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isPodUnhealthyForUpgrade(c.pod); got != c.want {
				t.Errorf("isPodUnhealthyForUpgrade = %v, want %v", got, c.want)
			}
		})
	}
}
