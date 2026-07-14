package docker

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/registry"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what fn wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	os.Stdout = orig
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(out)
}

func TestEncodeAuthToBase64(t *testing.T) {
	in := registry.AuthConfig{Username: "smurf", Password: "s3cr3t", ServerAddress: "registry.example.com"}
	enc, err := encodeAuthToBase64(in)
	if err != nil {
		t.Fatalf("encodeAuthToBase64: %v", err)
	}
	// The encoding must be URL-safe base64 (used verbatim in a registry auth header).
	raw, err := base64.URLEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("result is not URL-safe base64 (%q): %v", enc, err)
	}
	var got registry.AuthConfig
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decoded payload is not valid AuthConfig JSON: %v", err)
	}
	if got.Username != in.Username || got.Password != in.Password || got.ServerAddress != in.ServerAddress {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, in)
	}
}

func TestPrepareAuth(t *testing.T) {
	enc, err := prepareAuth("user", "pass", "server.io")
	if err != nil {
		t.Fatalf("prepareAuth: %v", err)
	}
	raw, err := base64.URLEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("result is not URL-safe base64 (%q): %v", enc, err)
	}
	var got registry.AuthConfig
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decoded payload is not valid AuthConfig JSON: %v", err)
	}
	if got.Username != "user" || got.Password != "pass" || got.ServerAddress != "server.io" {
		t.Errorf("fields mismatch: got %+v", got)
	}
}

func TestIsMeaningfulStatus(t *testing.T) {
	cases := []struct {
		status string
		want   bool
	}{
		{"Preparing", true},
		{"Pushing", true},
		{"Pushed", true},
		{"Layer already exists", true},
		{"Image", false},
		{"latest: digest", false},
		{"Mounted from cache", false},
		{"", true},
	}
	for _, c := range cases {
		if got := isMeaningfulStatus(c.status); got != c.want {
			t.Errorf("isMeaningfulStatus(%q) = %v, want %v", c.status, got, c.want)
		}
	}
}

func TestHasSimilarStatus(t *testing.T) {
	cases := []struct {
		name     string
		existing []string
		newSt    string
		want     bool
	}{
		{"exact match", []string{"Pushing"}, "Pushing", true},
		{"new contains existing", []string{"Push"}, "Pushing", true},
		{"existing contains new", []string{"Pushing"}, "Push", true},
		{"no overlap", []string{"Preparing"}, "Pushing", false},
		{"empty list", nil, "Pushing", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasSimilarStatus(c.existing, c.newSt); got != c.want {
				t.Errorf("hasSimilarStatus(%v, %q) = %v, want %v", c.existing, c.newSt, got, c.want)
			}
		})
	}
}

func TestExtractServerAddress(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"gcr.io/project/image:tag", "gcr.io"},
		{"registry.example.com/app", "registry.example.com"},
		{"nginx", "nginx"},
		{"", ""},
	}
	for _, c := range cases {
		if got := extractServerAddress(c.in); got != c.want {
			t.Errorf("extractServerAddress(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseImageNameAndTag(t *testing.T) {
	cases := []struct {
		in      string
		wantImg string
		wantTag string
	}{
		{"nginx:1.25", "nginx", "1.25"},
		{"nginx", "nginx", "latest"},
		{"host:5000/app", "host", "5000/app"},              // single colon (registry port) splits into two parts
		{"host:5000/app:v2", "host:5000/app:v2", "latest"}, // three colon-parts fall back to latest
	}
	for _, c := range cases {
		img, tag := parseImageNameAndTag(c.in)
		if img != c.wantImg || tag != c.wantTag {
			t.Errorf("parseImageNameAndTag(%q) = (%q, %q), want (%q, %q)", c.in, img, tag, c.wantImg, c.wantTag)
		}
	}
}

func TestColorHelpers(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) string
		code string
	}{
		{"green", green, "\033[32m"},
		{"red", red, "\033[31m"},
		{"cyan", cyan, "\033[36m"},
		{"blue", blue, "\033[34m"},
		{"magenta", magenta, "\033[35m"},
		{"bold", bold, "\033[1m"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.fn("hi")
			if !strings.HasPrefix(got, c.code) {
				t.Errorf("%s prefix = %q, want prefix %q", c.name, got, c.code)
			}
			if !strings.HasSuffix(got, "\033[0m") {
				t.Errorf("%s does not reset color: %q", c.name, got)
			}
			if !strings.Contains(got, "hi") {
				t.Errorf("%s dropped the message: %q", c.name, got)
			}
		})
	}
}

func TestNewStepTracker(t *testing.T) {
	st := newStepTracker(5)
	if st.total != 5 {
		t.Errorf("total = %d, want 5", st.total)
	}
	if st.current != 0 {
		t.Errorf("current = %d, want 0", st.current)
	}
	if st.start.IsZero() {
		t.Error("start time should be initialized")
	}
}

func TestDisplayLayerProgress(t *testing.T) {
	layerOrder := []string{"layer1"}
	layerStatus := map[string][]string{
		"layer1": {"Preparing", "Pushing", "Pushed"},
	}
	out := captureStdout(t, func() { displayLayerProgress(layerOrder, layerStatus) })
	for _, want := range []string{"Layer 1: layer1", "Preparing", "Uploading", "Pushed"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
}

func TestIsWindows(t *testing.T) {
	// The test host is darwin/linux in CI, so this must report false.
	if (&AuthProvider{}).isWindows() {
		t.Error("isWindows() = true on a non-Windows test host")
	}
}

func TestIsSafeBinary(t *testing.T) {
	ap := &AuthProvider{}

	t.Run("relative path is rejected", func(t *testing.T) {
		if ap.isSafeBinary("bin/terraform") {
			t.Error("expected relative path to be unsafe")
		}
	})

	t.Run("nonexistent absolute path is rejected", func(t *testing.T) {
		if ap.isSafeBinary("/definitely/not/here/binary") {
			t.Error("expected nonexistent path to be unsafe")
		}
	})

	t.Run("directory is rejected", func(t *testing.T) {
		dir := t.TempDir()
		if ap.isSafeBinary(dir) {
			t.Error("expected a directory to be unsafe")
		}
	})

	t.Run("world-writable file is rejected", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "tool")
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		// Chmod applies the mode directly, bypassing the process umask that
		// would otherwise strip the world-writable bit from WriteFile.
		if err := os.Chmod(p, 0o777); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		if ap.isSafeBinary(p) {
			t.Error("expected a world-writable file to be unsafe")
		}
	})

	t.Run("regular non-writable file in a clean path is safe", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "tool")
		if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
			t.Fatalf("write: %v", err)
		}
		// isSafeBinary rejects paths containing certain substrings (e.g. /tmp/);
		// skip the positive assertion if this host's temp dir happens to match one.
		lower := strings.ToLower(filepath.Clean(p))
		for _, bad := range []string{"/tmp/", "/var/tmp/", "/dev/", "/proc/", "..", "./", "~"} {
			if strings.Contains(lower, bad) {
				t.Skipf("temp path %q contains reserved substring %q", p, bad)
			}
		}
		if !ap.isSafeBinary(p) {
			t.Errorf("expected %q to be safe", p)
		}
	})
}
