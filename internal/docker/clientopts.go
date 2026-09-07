package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/docker/docker/client"
)

// The Docker SDK does not read Docker CLI contexts. With DOCKER_HOST unset it
// falls back to unix:///var/run/docker.sock, and on Docker Desktop for macOS
// that path does not exist: the active context points at
// ~/.docker/run/docker.sock instead, and recent versions leave the
// "allow the default Docker socket" option off.
//
// The result was that every sdkr command reported
//
//	Cannot connect to the Docker daemon at unix:///var/run/docker.sock
//
// on a machine where `docker` itself worked perfectly. Someone hitting that
// concludes smurf is broken, and they are not wrong.
//
// Resolving the context the way the CLI does fixes it, and also picks up
// colima, Rancher Desktop and remote contexts, which have the same shape.

// dockerConfigDir returns the Docker CLI configuration directory.
func dockerConfigDir() string {
	if dir := os.Getenv("DOCKER_CONFIG"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".docker")
}

// currentContextName reads the selected context from the CLI configuration.
// DOCKER_CONTEXT wins, matching the CLI's own precedence.
func currentContextName() string {
	if name := os.Getenv("DOCKER_CONTEXT"); name != "" {
		return name
	}
	dir := dockerConfigDir()
	if dir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json")) // #nosec G304 G703 -- path derived from the user's own Docker config dir
	if err != nil {
		return ""
	}
	var cfg struct {
		CurrentContext string `json:"currentContext"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return ""
	}
	return cfg.CurrentContext
}

// contextHost returns the docker endpoint for a named context.
//
// The CLI stores each context under a directory named for the sha256 of the
// context name, which is why this hashes rather than searching.
func contextHost(name string) string {
	if name == "" || name == "default" {
		return ""
	}
	dir := dockerConfigDir()
	if dir == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(name))
	meta := filepath.Join(dir, "contexts", "meta", hex.EncodeToString(sum[:]), "meta.json")

	data, err := os.ReadFile(meta) // #nosec G304 G703 -- path derived from the user's own Docker config dir
	if err != nil {
		return ""
	}
	var m struct {
		Endpoints struct {
			Docker struct {
				Host string `json:"Host"`
			} `json:"docker"`
		} `json:"Endpoints"`
	}
	if json.Unmarshal(data, &m) != nil {
		return ""
	}
	return m.Endpoints.Docker.Host
}

// resolveDockerHost returns the endpoint smurf should connect to, or an empty
// string to leave the SDK's own default in place.
//
// DOCKER_HOST is honoured first and returns empty, because client.FromEnv
// already handles it and overriding would only risk disagreeing with it.
func resolveDockerHost() string {
	if os.Getenv("DOCKER_HOST") != "" {
		return ""
	}
	return contextHost(currentContextName())
}

// clientOpts returns the options every Docker client in this package should be
// built with.
//
// WithAPIVersionNegotiation matters as much as the host: without it the client
// sends its compiled-in API version and a daemon older than that refuses the
// request outright.
func clientOpts() []client.Opt {
	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if host := resolveDockerHost(); host != "" {
		opts = append(opts, client.WithHost(host))
	}
	return opts
}
