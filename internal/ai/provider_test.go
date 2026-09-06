package ai

import (
	"strings"
	"testing"
)

func TestResolveProvider_ReadsEnvironment(t *testing.T) {
	t.Setenv(envAPIKey, "sk-test")
	t.Setenv(envBaseURL, "https://api.groq.com/openai/v1")
	t.Setenv(envModel, "llama-3.3-70b")

	p := resolveProvider()

	if p.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want sk-test", p.APIKey)
	}
	if p.BaseURL != "https://api.groq.com/openai/v1" {
		t.Errorf("BaseURL = %q", p.BaseURL)
	}
	if p.Model != "llama-3.3-70b" {
		t.Errorf("Model = %q", p.Model)
	}
}

func TestResolveProvider_TrimsWhitespace(t *testing.T) {
	// A key pasted from a dashboard often carries a trailing newline, which
	// would otherwise be sent in the Authorization header and rejected with an
	// opaque 401.
	t.Setenv(envAPIKey, "  sk-test\n")
	t.Setenv(envBaseURL, " http://localhost:11434/v1 ")

	p := resolveProvider()

	if p.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want the trimmed value", p.APIKey)
	}
	if p.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("BaseURL = %q, want the trimmed value", p.BaseURL)
	}
}

func TestIsLocalEndpoint(t *testing.T) {
	local := []string{
		"http://localhost:11434/v1",
		"http://127.0.0.1:8080/v1",
		"http://[::1]:11434/v1",
		"http://0.0.0.0:11434/v1",
		"http://host.docker.internal:11434/v1",
		"HTTP://LOCALHOST:11434/V1",
	}
	for _, url := range local {
		if !isLocalEndpoint(url) {
			t.Errorf("isLocalEndpoint(%q) = false, want true", url)
		}
	}

	remote := []string{
		"",
		"https://api.openai.com/v1",
		"https://api.groq.com/openai/v1",
		"https://openrouter.ai/api/v1",
	}
	for _, url := range remote {
		if isLocalEndpoint(url) {
			t.Errorf("isLocalEndpoint(%q) = true, want false", url)
		}
	}
}

// A local server is the configuration that costs nothing, so requiring a key
// there would block the only free path.
func TestCredentialsMissing_LocalNeedsNoKey(t *testing.T) {
	p := providerConfig{BaseURL: "http://localhost:11434/v1"}

	missing, _ := p.credentialsMissing()

	if missing {
		t.Error("a local endpoint should not require an API key")
	}
}

func TestCredentialsMissing_RemoteNeedsKey(t *testing.T) {
	for _, p := range []providerConfig{
		{}, // default OpenAI, no key
		{BaseURL: "https://api.groq.com/openai/v1"}, // custom remote, no key
	} {
		missing, guidance := p.credentialsMissing()
		if !missing {
			t.Errorf("%+v should require a key", p)
		}
		if guidance == "" {
			t.Errorf("%+v should explain what to do", p)
		}
	}
}

// The guidance is the only thing a user sees when AI silently does nothing, so
// it has to name a concrete way forward rather than just the problem.
func TestCredentialsMissing_GuidanceOffersAFreeOption(t *testing.T) {
	missing, guidance := providerConfig{}.credentialsMissing()

	if !missing {
		t.Fatal("expected credentials to be missing")
	}
	if !strings.Contains(guidance, envBaseURL) {
		t.Errorf("guidance should mention %s as an alternative:\n%s", envBaseURL, guidance)
	}
	if !strings.Contains(guidance, "localhost:11434") {
		t.Errorf("guidance should point at a local server that costs nothing:\n%s", guidance)
	}
}

func TestCredentialsMissing_KeyPresentIsFine(t *testing.T) {
	missing, _ := providerConfig{APIKey: "sk-test"}.credentialsMissing()
	if missing {
		t.Error("a provider with a key should not report missing credentials")
	}
}

// Some OpenAI-compatible servers reject an empty Authorization header even
// when they ignore its contents.
func TestAuthToken_PlaceholderWhenNoKey(t *testing.T) {
	if got := (providerConfig{APIKey: "sk-real"}).authToken(); got != "sk-real" {
		t.Errorf("authToken() = %q, want the real key", got)
	}
	if got := (providerConfig{}).authToken(); got == "" {
		t.Error("authToken() should return a placeholder rather than an empty string")
	}
}
