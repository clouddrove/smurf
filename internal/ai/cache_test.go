package ai

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// useTempCache points the cache at a temp directory for the duration of a test.
// os.UserCacheDir honours XDG_CACHE_HOME on Linux and HOME on macOS, so both
// are set to keep this working on either.
func useTempCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv(envNoCache, "")
	t.Setenv(envCacheTTL, "")
	return dir
}

func TestCacheRoundTrip(t *testing.T) {
	useTempCache(t)
	p := providerConfig{Model: "gpt-4o-mini"}

	if _, ok := cacheLookup(p, "prompt"); ok {
		t.Fatal("expected a miss on an empty cache")
	}

	cacheStore(p, "prompt", "an explanation")

	got, ok := cacheLookup(p, "prompt")
	if !ok {
		t.Fatal("expected a hit after storing")
	}
	if got != "an explanation" {
		t.Errorf("cached value = %q", got)
	}
}

// A stale explanation from a different model would be actively misleading, so
// the model and endpoint are part of the identity of an entry.
func TestCacheKeyVariesByModelAndEndpoint(t *testing.T) {
	base := providerConfig{Model: "gpt-4o-mini"}
	otherModel := providerConfig{Model: "llama3.2"}
	otherHost := providerConfig{Model: "gpt-4o-mini", BaseURL: "http://localhost:11434/v1"}

	k1 := cacheKey(base, "same prompt")
	k2 := cacheKey(otherModel, "same prompt")
	k3 := cacheKey(otherHost, "same prompt")
	k4 := cacheKey(base, "different prompt")

	for name, other := range map[string]string{"model": k2, "endpoint": k3, "prompt": k4} {
		if k1 == other {
			t.Errorf("cache key should change with %s", name)
		}
	}
	if k1 != cacheKey(base, "same prompt") {
		t.Error("cache key should be stable for identical inputs")
	}
}

func TestCacheRespectsTTL(t *testing.T) {
	useTempCache(t)
	p := providerConfig{Model: "gpt-4o-mini"}
	cacheStore(p, "prompt", "an explanation")

	// One second, then age the entry past it by backdating the file rather
	// than sleeping, so the test stays fast and deterministic.
	t.Setenv(envCacheTTL, "1")
	path := filepath.Join(cacheDir(), cacheKey(p, "prompt"))
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("could not backdate the cache entry: %v", err)
	}

	if _, ok := cacheLookup(p, "prompt"); ok {
		t.Error("an entry older than the TTL should not be served")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("an expired entry should be evicted")
	}
}

func TestCacheCanBeDisabled(t *testing.T) {
	useTempCache(t)
	p := providerConfig{Model: "gpt-4o-mini"}

	t.Setenv(envNoCache, "1")
	cacheStore(p, "prompt", "an explanation")
	if _, ok := cacheLookup(p, "prompt"); ok {
		t.Error("no lookup should succeed while the cache is disabled")
	}

	t.Setenv(envNoCache, "")
	if _, ok := cacheLookup(p, "prompt"); ok {
		t.Error("nothing should have been written while the cache was disabled")
	}
}

func TestCacheTTLParsing(t *testing.T) {
	cases := map[string]time.Duration{
		"":         defaultCacheTTL,
		"30":       30 * time.Second,
		"2h":       2 * time.Hour,
		"nonsense": defaultCacheTTL,
		"-5":       defaultCacheTTL, // a negative TTL would expire everything instantly
		"0":        defaultCacheTTL,
	}
	for raw, want := range cases {
		t.Setenv(envCacheTTL, raw)
		if got := cacheTTL(); got != want {
			t.Errorf("cacheTTL(%q) = %v, want %v", raw, got, want)
		}
	}
}

// An explanation embeds the error text it came from, which can carry paths and
// infrastructure detail, so it should not be world readable.
func TestCacheFilesAreNotWorldReadable(t *testing.T) {
	useTempCache(t)
	p := providerConfig{Model: "gpt-4o-mini"}
	cacheStore(p, "prompt", "an explanation")

	info, err := os.Stat(filepath.Join(cacheDir(), cacheKey(p, "prompt")))
	if err != nil {
		t.Fatalf("cache entry not written: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("cache file mode = %v, want no group or other access", perm)
	}
}
