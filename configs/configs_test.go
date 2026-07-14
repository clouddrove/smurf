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
	// Intentionally leave TEST_SMURF_MISSING_VAR unset: missing braced vars
	// expand to an empty string.
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
	// Missing braced env vars expand to empty string.
	if cfg.Sdkr.GithubToken != "" {
		t.Errorf("Sdkr.GithubToken = %q, want empty string for unset env var", cfg.Sdkr.GithubToken)
	}
}

func TestLoadConfig_LiteralDollarPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "smurf.yaml")

	// Passwords with literal $ characters must survive loading untouched:
	// only the braced ${VAR} form is interpolated.
	content := `
sdkr:
  docker_password: P@ss$word123
  github_token: abc$!def
  awsSecretKey: $TEST_SMURF_BARE_VAR
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Even if a same-named env var exists, the bare form must NOT be expanded.
	t.Setenv("TEST_SMURF_BARE_VAR", "should-not-appear")
	t.Setenv("word123", "should-not-appear-either")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Sdkr.DockerPassword != "P@ss$word123" {
		t.Errorf("Sdkr.DockerPassword = %q, want literal %q preserved", cfg.Sdkr.DockerPassword, "P@ss$word123")
	}
	if cfg.Sdkr.GithubToken != "abc$!def" {
		t.Errorf("Sdkr.GithubToken = %q, want literal %q preserved", cfg.Sdkr.GithubToken, "abc$!def")
	}
	if cfg.Sdkr.AwsSecretKey != "$TEST_SMURF_BARE_VAR" {
		t.Errorf("Sdkr.AwsSecretKey = %q, want bare $VAR left unexpanded as %q", cfg.Sdkr.AwsSecretKey, "$TEST_SMURF_BARE_VAR")
	}
}

func TestExpandBracedEnv(t *testing.T) {
	t.Setenv("TEST_SMURF_SET_VAR", "value")
	os.Unsetenv("TEST_SMURF_UNSET_VAR")

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"braced set var expanded", "prefix-${TEST_SMURF_SET_VAR}-suffix", "prefix-value-suffix"},
		{"braced missing var becomes empty", "prefix-${TEST_SMURF_UNSET_VAR}-suffix", "prefix--suffix"},
		{"bare var not expanded", "$TEST_SMURF_SET_VAR", "$TEST_SMURF_SET_VAR"},
		{"literal dollar preserved", "P@ss$word123", "P@ss$word123"},
		{"dollar with punctuation preserved", "abc$!def", "abc$!def"},
		{"trailing dollar preserved", "cost$", "cost$"},
		{"unterminated brace preserved", "${TEST_SMURF_SET_VAR", "${TEST_SMURF_SET_VAR"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := expandBracedEnv(tc.input); got != tc.want {
				t.Errorf("expandBracedEnv(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
