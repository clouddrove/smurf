package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// writeContext lays out a Docker CLI config directory the way the CLI does:
// config.json naming the current context, and each context stored under a
// directory named for the sha256 of its name.
func writeContext(t *testing.T, name, host string) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "config.json"),
		[]byte(`{"currentContext":"`+name+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256([]byte(name))
	metaDir := filepath.Join(dir, "contexts", "meta", hex.EncodeToString(sum[:]))
	if err := os.MkdirAll(metaDir, 0o700); err != nil {
		t.Fatal(err)
	}
	meta := `{"Name":"` + name + `","Endpoints":{"docker":{"Host":"` + host + `"}}}`
	if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(meta), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The bug this fixes: on Docker Desktop for macOS the active context points at
// ~/.docker/run/docker.sock while the SDK defaults to /var/run/docker.sock,
// which does not exist. Every sdkr command failed with "Cannot connect to the
// Docker daemon" on a machine where docker itself worked.
func TestResolveDockerHost_UsesTheActiveContext(t *testing.T) {
	dir := writeContext(t, "desktop-linux", "unix:///Users/someone/.docker/run/docker.sock")
	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")

	if got := resolveDockerHost(); got != "unix:///Users/someone/.docker/run/docker.sock" {
		t.Errorf("resolveDockerHost() = %q, want the context's endpoint", got)
	}
}

// DOCKER_HOST is handled by client.FromEnv already; returning a value here as
// well could only disagree with it.
func TestResolveDockerHost_DeferToDockerHost(t *testing.T) {
	dir := writeContext(t, "desktop-linux", "unix:///from/context.sock")
	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_HOST", "tcp://192.168.1.10:2375")

	if got := resolveDockerHost(); got != "" {
		t.Errorf("resolveDockerHost() = %q, want empty so FromEnv wins", got)
	}
}

// DOCKER_CONTEXT overrides the configured context, matching the CLI.
func TestResolveDockerHost_DockerContextOverrides(t *testing.T) {
	dir := writeContext(t, "desktop-linux", "unix:///desktop.sock")

	// Add a second context and select it by environment.
	sum := sha256.Sum256([]byte("colima"))
	metaDir := filepath.Join(dir, "contexts", "meta", hex.EncodeToString(sum[:]))
	if err := os.MkdirAll(metaDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, "meta.json"),
		[]byte(`{"Name":"colima","Endpoints":{"docker":{"Host":"unix:///colima.sock"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "colima")

	if got := resolveDockerHost(); got != "unix:///colima.sock" {
		t.Errorf("resolveDockerHost() = %q, want the context named by DOCKER_CONTEXT", got)
	}
}

// The "default" context means the SDK's own default, so returning nothing is
// correct and lets FromEnv decide.
func TestResolveDockerHost_DefaultContextChangesNothing(t *testing.T) {
	dir := writeContext(t, "default", "unix:///should-be-ignored.sock")
	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")

	if got := resolveDockerHost(); got != "" {
		t.Errorf("resolveDockerHost() = %q, want empty for the default context", got)
	}
}

// A machine with no Docker config at all must still work, falling back to the
// SDK default rather than erroring.
func TestResolveDockerHost_NoConfigIsSafe(t *testing.T) {
	t.Setenv("DOCKER_CONFIG", t.TempDir())
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")

	if got := resolveDockerHost(); got != "" {
		t.Errorf("resolveDockerHost() = %q, want empty with no config", got)
	}
}

// Malformed JSON should not take a command down; the SDK default is a fine
// outcome.
func TestResolveDockerHost_MalformedConfigIsSafe(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")

	if got := resolveDockerHost(); got != "" {
		t.Errorf("resolveDockerHost() = %q, want empty for unreadable config", got)
	}
}

// Version negotiation has to survive this change: without it the client sends
// its compiled-in API version and a daemon older than that refuses outright,
// which is a separate failure this package already had to fix once.
func TestClientOpts_AlwaysIncludesEnvAndNegotiation(t *testing.T) {
	t.Setenv("DOCKER_CONFIG", t.TempDir())
	t.Setenv("DOCKER_HOST", "")

	if got := len(clientOpts()); got < 2 {
		t.Errorf("clientOpts() returned %d options, want at least FromEnv and version negotiation", got)
	}

	dir := writeContext(t, "desktop-linux", "unix:///ctx.sock")
	t.Setenv("DOCKER_CONFIG", dir)
	if got := len(clientOpts()); got < 3 {
		t.Errorf("clientOpts() returned %d options, want the host added when a context resolves", got)
	}
}
