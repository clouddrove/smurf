package ai

import (
	"os"
	"strings"
)

// Provider settings resolved from the environment.
//
// Almost every hosted model service now speaks the OpenAI chat-completions
// protocol, so pointing the existing client at a different base URL is enough
// to reach Groq, OpenRouter, DeepSeek, Together, Gemini's compatibility
// endpoint, or a local Ollama or llama.cpp server. That covers both bring your
// own key and running at no cost, without a second SDK or a provider registry
// to maintain.

// These name environment variables; they hold no secret themselves.
const (
	envAuthVar = "OPENAI_API_KEY"
	envBaseURL = "OPENAI_BASE_URL"
	envModel   = "OPENAI_MODEL"
)

// providerConfig is the resolved endpoint for a single call.
type providerConfig struct {
	APIKey  string
	BaseURL string // empty means the OpenAI default
	Model   string
}

// resolveProvider reads provider settings from the environment.
//
// A local server is the reason APIKey may legitimately be empty: Ollama and
// llama.cpp accept any token, so demanding a real key would block the one
// configuration that costs nothing. The rule is therefore that a key is
// required only when talking to the default OpenAI endpoint.
func resolveProvider() providerConfig {
	return providerConfig{
		APIKey:  strings.TrimSpace(os.Getenv(envAuthVar)),
		BaseURL: strings.TrimSpace(os.Getenv(envBaseURL)),
		Model:   modelFromEnv(),
	}
}

// isLocalEndpoint reports whether the base URL points at the machine running
// smurf. Local servers are the case where a missing key is expected rather
// than a misconfiguration.
func isLocalEndpoint(baseURL string) bool {
	if baseURL == "" {
		return false
	}
	lowered := strings.ToLower(baseURL)
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]", "0.0.0.0", "host.docker.internal"} {
		if strings.Contains(lowered, host) {
			return true
		}
	}
	return false
}

// credentialsMissing reports whether the resolved provider cannot authenticate,
// along with a message naming what to do about it. Keeping the decision here
// rather than in IsEnabled lets it be tested without touching the terminal.
func (p providerConfig) credentialsMissing() (bool, string) {
	if p.APIKey != "" {
		return false, ""
	}
	if isLocalEndpoint(p.BaseURL) {
		// A local server does not check the token, so this is fine.
		return false, ""
	}
	if p.BaseURL != "" {
		return true, "AI mode enabled but no " + envAuthVar + " found for " + p.BaseURL +
			"\nSet a key for that endpoint, or point " + envBaseURL + " at a local server such as http://localhost:11434/v1"
	}
	return true, "AI mode enabled but no " + envAuthVar + " found." +
		"\nEither export a key, or set " + envBaseURL + " to another OpenAI-compatible endpoint" +
		"\n  free and local:  export " + envBaseURL + "=http://localhost:11434/v1 && export " + envModel + "=llama3.2" +
		"\n  hosted free tier: export " + envBaseURL + "=https://api.groq.com/openai/v1 && export " + envAuthVar + "=<key>"
}

// authToken is what gets handed to the client. Local servers ignore the value
// but the SDK still sends an Authorization header, and some servers reject an
// empty one, so a placeholder is used rather than nothing.
func (p providerConfig) authToken() string {
	if p.APIKey != "" {
		return p.APIKey
	}
	return "no-key-required"
}
