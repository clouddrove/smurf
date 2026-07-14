package configs

import (
	"os"
	"testing"
)

func TestExportEnvironmentVariables(t *testing.T) {
	t.Run("sets every variable from the map", func(t *testing.T) {
		keys := []string{"TEST_SMURF_EXPORT_ONE", "TEST_SMURF_EXPORT_TWO"}
		t.Cleanup(func() {
			for _, k := range keys {
				os.Unsetenv(k)
			}
		})

		vars := map[string]string{
			keys[0]: "alpha",
			keys[1]: "beta",
		}
		if err := ExportEnvironmentVariables(vars); err != nil {
			t.Fatalf("ExportEnvironmentVariables returned error: %v", err)
		}
		if got := os.Getenv(keys[0]); got != "alpha" {
			t.Errorf("%s = %q, want %q", keys[0], got, "alpha")
		}
		if got := os.Getenv(keys[1]); got != "beta" {
			t.Errorf("%s = %q, want %q", keys[1], got, "beta")
		}
	})

	t.Run("empty map is a no-op and succeeds", func(t *testing.T) {
		if err := ExportEnvironmentVariables(map[string]string{}); err != nil {
			t.Errorf("ExportEnvironmentVariables(empty) returned error: %v", err)
		}
	})

	t.Run("empty value is allowed", func(t *testing.T) {
		key := "TEST_SMURF_EXPORT_EMPTY"
		t.Cleanup(func() { os.Unsetenv(key) })
		if err := ExportEnvironmentVariables(map[string]string{key: ""}); err != nil {
			t.Fatalf("ExportEnvironmentVariables returned error: %v", err)
		}
		got, ok := os.LookupEnv(key)
		if !ok {
			t.Fatalf("%s was not set", key)
		}
		if got != "" {
			t.Errorf("%s = %q, want empty string", key, got)
		}
	})
}
