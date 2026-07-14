package utils

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidOutputFormat(t *testing.T) {
	cases := []struct {
		name    string
		format  string
		allowed []string
		want    bool
	}{
		{"match first", "json", []string{"json", "yaml"}, true},
		{"match later", "yaml", []string{"json", "yaml"}, true},
		{"no match", "xml", []string{"json", "yaml"}, false},
		{"empty format", "", []string{"json", "yaml"}, false},
		{"empty allowed", "json", nil, false},
		{"case sensitive", "JSON", []string{"json"}, false},
		{"empty format in allowed", "", []string{"json", ""}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ValidOutputFormat(c.format, c.allowed...); got != c.want {
				t.Errorf("ValidOutputFormat(%q, %v) = %v, want %v", c.format, c.allowed, got, c.want)
			}
		})
	}
}

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

func TestPrintJSON(t *testing.T) {
	t.Run("valid value round-trips and is the only output", func(t *testing.T) {
		in := map[string]any{"name": "smurf", "count": 3}
		var retErr error
		out := captureStdout(t, func() { retErr = PrintJSON(in) })
		if retErr != nil {
			t.Fatalf("PrintJSON returned error: %v", retErr)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("stdout is not valid JSON (%q): %v", out, err)
		}
		if got["name"] != "smurf" {
			t.Errorf("name = %v, want smurf", got["name"])
		}
		// indented output ends with a single trailing newline from Println
		if !strings.HasSuffix(out, "}\n") {
			t.Errorf("expected a single trailing newline after the JSON, got %q", out)
		}
	})

	t.Run("unmarshalable value returns a wrapped error", func(t *testing.T) {
		var retErr error
		_ = captureStdout(t, func() { retErr = PrintJSON(make(chan int)) })
		if retErr == nil {
			t.Fatal("expected an error for an unmarshalable value, got nil")
		}
		if !strings.Contains(retErr.Error(), "json marshal error") {
			t.Errorf("error %q does not mention json marshal", retErr.Error())
		}
	})
}

func TestCreateYamlFile(t *testing.T) {
	t.Run("creates a 0600 file with the given content", func(t *testing.T) {
		t.Chdir(t.TempDir())
		_ = captureStdout(t, func() {
			if err := CreateYamlFile("smurf.yaml", "key: value\n"); err != nil {
				t.Fatalf("CreateYamlFile: %v", err)
			}
		})
		got, err := os.ReadFile("smurf.yaml")
		if err != nil {
			t.Fatalf("read back: %v", err)
		}
		if string(got) != "key: value\n" {
			t.Errorf("content = %q, want %q", string(got), "key: value\n")
		}
		info, err := os.Stat("smurf.yaml")
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("perm = %o, want 600 (may hold credentials)", perm)
		}
	})

	t.Run("refuses to overwrite an existing file", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.WriteFile(filepath.Join(dir, "smurf.yaml"), []byte("existing"), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
		err := CreateYamlFile("smurf.yaml", "new content")
		if err == nil {
			t.Fatal("expected an error when the file already exists, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error %q does not mention 'already exists'", err.Error())
		}
		// the original content must be untouched
		got, _ := os.ReadFile(filepath.Join(dir, "smurf.yaml"))
		if string(got) != "existing" {
			t.Errorf("existing file was modified: %q", string(got))
		}
	})
}
