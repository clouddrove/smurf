package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubEndpoint stands up a minimal OpenAI-compatible server. Real free options
// such as Ollama, llama.cpp and Groq speak this same protocol, so exercising it
// here proves the client actually talks to a non-OpenAI endpoint rather than
// merely being configured to.
func stubEndpoint(t *testing.T, reply string, capture *map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capture != nil {
			body, _ := io.ReadAll(r.Body)
			var parsed map[string]any
			_ = json.Unmarshal(body, &parsed)
			parsed["_authorization"] = r.Header.Get("Authorization")
			parsed["_path"] = r.URL.Path
			*capture = parsed
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "stub", "object": "chat.completion", "model": "stub-model",
			"choices": [{"index":0,"message":{"role":"assistant","content":` +
			strconvQuote(reply) + `},"finish_reason":"stop"}]
		}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// The point of the provider work: a local endpoint with no API key must work,
// because that is the configuration that costs nothing.
func TestAskAI_AgainstLocalEndpointWithoutAKey(t *testing.T) {
	useTempCache(t)
	var seen map[string]any
	srv := stubEndpoint(t, "ROOT CAUSE:\n- stub", &seen)

	t.Setenv(envAuthVar, "")
	t.Setenv(envBaseURL, srv.URL+"/v1")
	t.Setenv(envModel, "llama3.2")

	got, err := AskAI("why did this fail?")
	if err != nil {
		t.Fatalf("AskAI against a local endpoint failed: %v", err)
	}
	if !strings.Contains(got, "stub") {
		t.Errorf("response = %q", got)
	}

	if seen["model"] != "llama3.2" {
		t.Errorf("model sent = %v, want llama3.2", seen["model"])
	}
	if path, _ := seen["_path"].(string); !strings.HasSuffix(path, "/chat/completions") {
		t.Errorf("request path = %v, want the chat completions endpoint", seen["_path"])
	}
	// Some servers reject an empty Authorization header even when they ignore
	// the value, so one must still be sent.
	if auth, _ := seen["_authorization"].(string); auth == "" {
		t.Error("an Authorization header should still be sent to a keyless endpoint")
	}
}

// Cost controls are only real if they reach the wire.
func TestAskAI_SendsCostControls(t *testing.T) {
	useTempCache(t)
	var seen map[string]any
	srv := stubEndpoint(t, "ok", &seen)

	t.Setenv(envAuthVar, "")
	t.Setenv(envBaseURL, srv.URL+"/v1")

	if _, err := AskAI("prompt"); err != nil {
		t.Fatalf("AskAI: %v", err)
	}

	if mt, ok := seen["max_tokens"].(float64); !ok || int(mt) != maxResponseTokens {
		t.Errorf("max_tokens sent = %v, want %d", seen["max_tokens"], maxResponseTokens)
	}
	if temp, ok := seen["temperature"].(float64); !ok || temp > 0.5 {
		t.Errorf("temperature sent = %v, want a low value for reproducible diagnoses", seen["temperature"])
	}
}

// A repeated failure should cost one call, not one per re-run.
func TestAskAI_SecondIdenticalCallIsServedFromCache(t *testing.T) {
	useTempCache(t)

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"cached answer"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv(envAuthVar, "")
	t.Setenv(envBaseURL, srv.URL+"/v1")

	first, err := AskAI("identical prompt")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, err := AskAI("identical prompt")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if first != second {
		t.Errorf("cached response differs: %q vs %q", first, second)
	}
	if calls != 1 {
		t.Errorf("endpoint was called %d times, want 1 with the second served from cache", calls)
	}
}

// A different prompt must not be answered from another prompt's entry.
func TestAskAI_DifferentPromptBypassesCache(t *testing.T) {
	useTempCache(t)

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"index":0,"message":{"role":"assistant","content":"answer"},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv(envAuthVar, "")
	t.Setenv(envBaseURL, srv.URL+"/v1")

	if _, err := AskAI("first prompt"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := AskAI("second prompt"); err != nil {
		t.Fatalf("second: %v", err)
	}

	if calls != 2 {
		t.Errorf("endpoint called %d times, want 2", calls)
	}
}

// With a custom endpoint, an error saying "openai error" would send the reader
// to the wrong service.
func TestAskAI_ErrorNamesTheEndpoint(t *testing.T) {
	useTempCache(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	t.Setenv(envAuthVar, "")
	t.Setenv(envBaseURL, srv.URL+"/v1")

	_, err := AskAI("prompt")
	if err == nil {
		t.Fatal("expected an error from a failing endpoint")
	}
	if strings.Contains(err.Error(), "openai") {
		t.Errorf("error should name the configured endpoint, not openai: %v", err)
	}
	if !strings.Contains(err.Error(), srv.URL) {
		t.Errorf("error should name %s, got: %v", srv.URL, err)
	}
}
