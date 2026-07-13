package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "smurf.yaml")

	_, err := LoadConfig(missing)
	if err == nil {
		t.Fatal("expected an error for a missing config file, got nil")
	}

	wantSubstr := "Pass flags explicitly or run 'smurf init' to create one"
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Errorf("error message = %q, want it to contain %q", err.Error(), wantSubstr)
	}
}

func TestLoadConfig_NormalLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smurf.yaml")

	content := `
sdkr:
  docker_username: myuser
  imageName: myimage
selm:
  releaseName: myrelease
  namespace: mynamespace
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Sdkr.DockerUsername != "myuser" {
		t.Errorf("Sdkr.DockerUsername = %q, want %q", cfg.Sdkr.DockerUsername, "myuser")
	}
	if cfg.Sdkr.ImageName != "myimage" {
		t.Errorf("Sdkr.ImageName = %q, want %q", cfg.Sdkr.ImageName, "myimage")
	}
	if cfg.Selm.ReleaseName != "myrelease" {
		t.Errorf("Selm.ReleaseName = %q, want %q", cfg.Selm.ReleaseName, "myrelease")
	}
	if cfg.Selm.Namespace != "mynamespace" {
		t.Errorf("Selm.Namespace = %q, want %q", cfg.Selm.Namespace, "mynamespace")
	}
}

func TestLoadConfig_EnvInterpolation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smurf.yaml")

	content := `
sdkr:
  docker_password: ${TEST_SMURF_DOCKER_PASSWORD}
  github_token: ${TEST_SMURF_MISSING_VAR}
selm:
  namespace: ${TEST_SMURF_NAMESPACE}
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	t.Setenv("TEST_SMURF_DOCKER_PASSWORD", "s3cr3t")
	t.Setenv("TEST_SMURF_NAMESPACE", "prod")
	// Intentionally leave TEST_SMURF_MISSING_VAR unset to exercise
	// os.ExpandEnv's default behavior: missing vars expand to "".
	os.Unsetenv("TEST_SMURF_MISSING_VAR")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Sdkr.DockerPassword != "s3cr3t" {
		t.Errorf("Sdkr.DockerPassword = %q, want %q", cfg.Sdkr.DockerPassword, "s3cr3t")
	}
	if cfg.Selm.Namespace != "prod" {
		t.Errorf("Selm.Namespace = %q, want %q", cfg.Selm.Namespace, "prod")
	}
	// Missing env vars expand to empty string, this is os.ExpandEnv's documented default.
	if cfg.Sdkr.GithubToken != "" {
		t.Errorf("Sdkr.GithubToken = %q, want empty string for unset env var", cfg.Sdkr.GithubToken)
	}
}
