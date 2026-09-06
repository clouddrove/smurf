package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Responses are cached on disk because the same failure tends to be explained
// repeatedly: a broken pipeline is re-run, a developer retries the same command
// after a partial fix, CI replays a job. Each of those was previously a fresh
// paid call returning the same text.
const (
	envNoCache  = "SMURF_AI_NO_CACHE"
	envCacheTTL = "SMURF_AI_CACHE_TTL"

	defaultCacheTTL = 24 * time.Hour
)

// cacheKey identifies a response by everything that can change it. The model
// and endpoint are included because the same prompt answered by a different
// model is a different answer, and a stale entry from another provider would
// be actively misleading.
func cacheKey(p providerConfig, prompt string) string {
	sum := sha256.Sum256([]byte(p.Model + "\x00" + p.BaseURL + "\x00" + prompt))
	return hex.EncodeToString(sum[:])
}

// cacheEnabled reports whether the on-disk cache should be consulted.
func cacheEnabled() bool {
	switch os.Getenv(envNoCache) {
	case "", "0", "false", "FALSE":
		return true
	default:
		return false
	}
}

// cacheTTL is how long an entry stays usable. Explanations age with the code
// and the provider's model, so they expire rather than persisting forever.
func cacheTTL() time.Duration {
	raw := os.Getenv(envCacheTTL)
	if raw == "" {
		return defaultCacheTTL
	}
	if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	return defaultCacheTTL
}

// cacheDir returns the directory holding cached responses, creating it if
// needed. An unavailable cache directory is not worth failing a command over,
// so callers treat the empty string as "caching off for this run".
func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(base, "smurf", "ai")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ""
	}
	return dir
}

// cacheLookup returns a cached response when one exists and is still fresh.
//
// Every failure path here returns a miss rather than an error: a cache that
// cannot be read should cost an API call, never a command.
func cacheLookup(p providerConfig, prompt string) (string, bool) {
	if !cacheEnabled() {
		return "", false
	}
	dir := cacheDir()
	if dir == "" {
		return "", false
	}
	path := filepath.Join(dir, cacheKey(p, prompt))

	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	if time.Since(info.ModTime()) > cacheTTL() {
		// Best-effort eviction; a failure here just means it is retried later.
		_ = os.Remove(path)
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return "", false
	}
	return string(data), true
}

// cacheStore writes a response to the cache, ignoring failures for the same
// reason lookups do.
func cacheStore(p providerConfig, prompt, response string) {
	if !cacheEnabled() || response == "" {
		return
	}
	dir := cacheDir()
	if dir == "" {
		return
	}
	// 0600: an explanation embeds the error text it was derived from, which can
	// include paths and infrastructure detail from the user's environment.
	_ = os.WriteFile(filepath.Join(dir, cacheKey(p, prompt)), []byte(response), 0o600)
}
