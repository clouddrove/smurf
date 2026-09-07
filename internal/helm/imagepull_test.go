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
