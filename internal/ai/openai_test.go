package ai

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what fn wrote.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	os.Stdout = orig
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(out)
}

// disableColor forces plain-text output from the color package so assertions
// can match on content without ANSI escape sequences.
func disableColor(t *testing.T) {
	t.Helper()
	prev := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = prev })
}

func TestModelFromEnv(t *testing.T) {
	t.Run("falls back to default when unset", func(t *testing.T) {
		os.Unsetenv("OPENAI_MODEL")
		if got := modelFromEnv(); got != string(defaultModel) {
			t.Errorf("modelFromEnv() = %q, want default %q", got, defaultModel)
		}
		if defaultModel != openai.GPT4oMini {
			t.Errorf("defaultModel = %q, want %q", defaultModel, openai.GPT4oMini)
		}
	})

	t.Run("uses the override from the environment", func(t *testing.T) {
		t.Setenv("OPENAI_MODEL", "gpt-4-custom")
		if got := modelFromEnv(); got != "gpt-4-custom" {
			t.Errorf("modelFromEnv() = %q, want %q", got, "gpt-4-custom")
		}
	})

	t.Run("empty override falls back to default", func(t *testing.T) {
		t.Setenv("OPENAI_MODEL", "")
		if got := modelFromEnv(); got != string(defaultModel) {
			t.Errorf("modelFromEnv() = %q, want default %q", got, defaultModel)
		}
	})
}

func TestIsEnabled(t *testing.T) {
	t.Run("true when key is present", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "sk-test-key")
		if !IsEnabled() {
			t.Error("IsEnabled() = false, want true when OPENAI_API_KEY is set")
		}
	})

	t.Run("false and warns when key is missing", func(t *testing.T) {
		disableColor(t)
		t.Setenv("OPENAI_API_KEY", "")

		// The color package writes to color.Output (captured at init), not the
		// live os.Stdout, so redirect it to observe the warning.
		var buf bytes.Buffer
		prev := color.Output
		color.Output = &buf
		t.Cleanup(func() { color.Output = prev })

		enabled := IsEnabled()
		if enabled {
			t.Error("IsEnabled() = true, want false when OPENAI_API_KEY is empty")
		}
		if !strings.Contains(buf.String(), "OPENAI_API_KEY") {
			t.Errorf("warning output = %q, want it to mention OPENAI_API_KEY", buf.String())
		}
	})
}

func TestAskAI_NoAPIKey(t *testing.T) {
	// With no key, AskAI must return an error before any network call is made.
	t.Setenv("OPENAI_API_KEY", "")
	_, err := AskAI("anything")
	if err == nil {
		t.Fatal("AskAI expected an error when OPENAI_API_KEY is unset, got nil")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY is not set") {
		t.Errorf("error = %q, want it to mention 'OPENAI_API_KEY is not set'", err.Error())
	}
}

func TestExplainError_NoAPIKey(t *testing.T) {
	// ExplainError redacts then calls AskAI; with no key it returns AskAI's error
	// without hitting the network.
	t.Setenv("OPENAI_API_KEY", "")
	out, err := ExplainError("some failure with ghp_1234567890abcdefABCDEF1234567890")
	if err == nil {
		t.Fatal("ExplainError expected an error when OPENAI_API_KEY is unset, got nil")
	}
	if out != "" {
		t.Errorf("ExplainError output = %q, want empty string on error", out)
	}
}

func TestAIExplainError(t *testing.T) {
	t.Run("no-op when useAI is false", func(t *testing.T) {
		out := captureStdout(t, func() { AIExplainError(false, "boom") })
		if strings.Contains(out, "Smurf AI Analysis") {
			t.Errorf("output = %q, want no analysis when useAI is false", out)
		}
	})

	t.Run("no analysis printed when AI is not enabled", func(t *testing.T) {
		disableColor(t)
		t.Setenv("OPENAI_API_KEY", "")

		var buf bytes.Buffer
		prev := color.Output
		color.Output = &buf
		t.Cleanup(func() { color.Output = prev })

		out := captureStdout(t, func() { AIExplainError(true, "boom") })
		if strings.Contains(out, "Smurf AI Analysis") {
			t.Errorf("output = %q, want no analysis header when no API key is present", out)
		}
	})
}
