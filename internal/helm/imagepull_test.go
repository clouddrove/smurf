package helm

import "testing"

// The registry states which problem it is, and the fixes differ: a wrong tag
// is corrected in the chart, missing credentials in an imagePullSecret, an
// unreachable registry somewhere else entirely. Reporting "invalid tag or
// missing credentials" sends a reader to check both.
func TestClassifyImagePullFailure(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    ImagePullCause
	}{
		{
			name:    "containerd not found",
			message: `failed to resolve reference "ghcr.io/acme/api:v2": ghcr.io/acme/api:v2: not found`,
			want:    ImagePullTagNotFound,
		},
		{
			name:    "docker hub manifest unknown",
			message: `Failed to pull image "acme/api:nope": manifest for acme/api:nope not found: manifest unknown`,
			want:    ImagePullTagNotFound,
		},
		{
			name:    "unauthorized",
			message: `Failed to pull image "ghcr.io/acme/private:v1": failed to authorize: unauthorized`,
			want:    ImagePullUnauthorized,
		},
		{
			name:    "denied",
			message: `pull access denied for acme/private, repository does not exist or may require authentication`,
			want:    ImagePullUnauthorized,
		},
		{
			name:    "docker hub rate limit",
			message: `toomanyrequests: You have reached your pull rate limit`,
			want:    ImagePullRateLimited,
		},
		{
			name:    "registry unreachable",
			message: `dial tcp: lookup registry.internal on 10.0.0.1:53: no such host`,
			want:    ImagePullRegistryDown,
		},
		{
			name:    "nothing definite",
			message: `Back-off pulling image`,
			want:    ImagePullCauseUnknown,
		},
		{
			name:    "empty",
			message: "",
			want:    ImagePullCauseUnknown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyImagePullFailure(tc.message); got != tc.want {
				t.Errorf("classifyImagePullFailure() = %q, want %q", got, tc.want)
			}
		})
	}
}

// An unauthorized response usually also names the reference that failed, and
// "repository does not exist or may require authentication" contains both
// phrasings. Checking not-found first would report an auth problem as a
// missing tag and send someone to edit the chart instead of the credentials.
func TestClassifyImagePullFailure_AuthWinsOverNotFound(t *testing.T) {
	message := `pull access denied for acme/private, repository does not exist or may require authentication`

	if got := classifyImagePullFailure(message); got != ImagePullUnauthorized {
		t.Errorf("classifyImagePullFailure() = %q, want the credentials cause", got)
	}
}

func TestImageFromPullMessage(t *testing.T) {
	msg := `Failed to pull image "ghcr.io/acme/api:v2.3.1": not found`
	if got := imageFromPullMessage(msg); got != "ghcr.io/acme/api:v2.3.1" {
		t.Errorf("imageFromPullMessage() = %q", got)
	}
}

// An unquoted message should yield nothing rather than a wrong guess, since a
// wrong image reference in the report is worse than none.
func TestImageFromPullMessage_NoQuotesYieldsNothing(t *testing.T) {
	for _, msg := range []string{"", "no quotes here", `only one " quote`} {
		if got := imageFromPullMessage(msg); got != "" {
			t.Errorf("imageFromPullMessage(%q) = %q, want empty", msg, got)
		}
	}
}

// The whole point: a definite sentence naming the image and the real cause.
func TestDescribeImagePullFailure(t *testing.T) {
	msg := `failed to resolve reference "ghcr.io/acme/api:v9": ghcr.io/acme/api:v9: not found`

	got := describeImagePullFailure(imageFromPullMessage(msg), msg)
	want := "image ghcr.io/acme/api:v9: image or tag does not exist in the registry"

	if got != want {
		t.Errorf("describeImagePullFailure() = %q, want %q", got, want)
	}
}

func TestDescribeImagePullFailure_UnknownStaysSilent(t *testing.T) {
	if got := describeImagePullFailure("acme/api:v1", "Back-off pulling image"); got != "" {
		t.Errorf("describeImagePullFailure() = %q, want empty when the cause is not stated", got)
	}
}

// Without the image the cause is still worth stating.
func TestDescribeImagePullFailure_NoImageStillReportsCause(t *testing.T) {
	got := describeImagePullFailure("", "unauthorized")
	if got != string(ImagePullUnauthorized) {
		t.Errorf("describeImagePullFailure() = %q, want the bare cause", got)
	}
}

// An upgrade whose only problem was a pod no node could schedule exited 0.
// Verified on a real cluster before the fix: a pod requesting more CPU than
// any node has left the upgrade reporting success.
func TestClassifyPendingBlocker(t *testing.T) {
	cases := []struct {
		name    string
		message string
		want    PendingBlocker
	}{
		{
			name:    "insufficient resources",
			message: `FailedScheduling: 0/2 nodes are available: 2 Insufficient cpu, 2 Insufficient memory.`,
			want:    PendingUnschedulable,
		},
		{
			name:    "node selector",
			message: `0/5 nodes are available: 5 node(s) didn't match node selector.`,
			want:    PendingUnschedulable,
		},
		{
			name:    "taints",
			message: `0/3 nodes are available: 3 node(s) had untolerated taint {dedicated: gpu}.`,
			want:    PendingUnschedulable,
		},
		{
			name:    "storage class missing",
			message: `storageclass.storage.k8s.io "no-such-storage-class" not found`,
			want:    PendingVolume,
		},
		{
			name:    "unbound claim",
			message: `pod has unbound immediate PersistentVolumeClaims`,
			want:    PendingVolume,
		},
		{
			name:    "quota",
			message: `pods "api-1" is forbidden: exceeded quota: compute, requested: cpu=2`,
			want:    PendingQuota,
		},
		{
			name:    "container creating is transient",
			message: `ContainerCreating`,
			want:    PendingBlockerNone,
		},
		{
			name:    "empty",
			message: "",
			want:    PendingBlockerNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyPendingBlocker(tc.message); got != tc.want {
				t.Errorf("classifyPendingBlocker() = %q, want %q", got, tc.want)
			}
		})
	}
}

// WaitForFirstConsumer volumes sit in exactly this state during a normal
// deployment and bind once the pod is scheduled. Treating it as a failure
// would break ordinary releases, which is a worse outcome than the bug being
// fixed here.
func TestClassifyPendingBlocker_WaitForFirstConsumerIsNormal(t *testing.T) {
	msg := `waiting for first consumer to be created before binding`

	if got := classifyPendingBlocker(msg); got != PendingBlockerNone {
		t.Errorf("classifyPendingBlocker() = %q, want no blocker; this is the normal state of a WaitForFirstConsumer volume", got)
	}
}

// A volume problem often also mentions scheduling, since the pod cannot be
// placed until the claim binds. Reporting it as unschedulable would send
// someone to look at node capacity instead of the storage class.
func TestClassifyPendingBlocker_VolumeWinsOverScheduling(t *testing.T) {
	msg := `0/2 nodes are available: 2 pod has unbound immediate PersistentVolumeClaims. preemption: not helpful`

	if got := classifyPendingBlocker(msg); got != PendingVolume {
		t.Errorf("classifyPendingBlocker() = %q, want the volume cause", got)
	}
}

func TestDescribePendingBlocker(t *testing.T) {
	if got := describePendingBlocker(`Insufficient cpu`); got != string(PendingUnschedulable) {
		t.Errorf("describePendingBlocker() = %q", got)
	}
	if got := describePendingBlocker(`ContainerCreating`); got != "" {
		t.Errorf("describePendingBlocker() = %q, want empty for a transient state", got)
	}
}
