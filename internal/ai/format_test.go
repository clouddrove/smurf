package ai

import (
	"strings"
	"testing"
)

func TestExtractSections(t *testing.T) {
	t.Run("parses multiple sections and continuation lines", func(t *testing.T) {
		response := strings.Join([]string{
			"ERROR ANALYSIS: something failed",
			"more detail on the failure",
			"ROOT CAUSE: the config was wrong",
			"STEPS TO RESOLVE:",
			"1. Fix the config",
			"2. Redeploy",
		}, "\n")

		sections := extractSections(response)

		if got := sections["ERROR ANALYSIS"]; !strings.Contains(got, "something failed") ||
			!strings.Contains(got, "more detail on the failure") {
			t.Errorf("ERROR ANALYSIS = %q, want inline + continuation content", got)
		}
		if got := sections["ROOT CAUSE"]; !strings.Contains(got, "the config was wrong") {
			t.Errorf("ROOT CAUSE = %q, want %q", got, "the config was wrong")
		}
		steps, ok := sections["STEPS TO RESOLVE"]
		if !ok {
			t.Fatal("STEPS TO RESOLVE section missing")
		}
		if !strings.Contains(steps, "Fix the config") || !strings.Contains(steps, "Redeploy") {
			t.Errorf("STEPS TO RESOLVE = %q, want the numbered steps", steps)
		}
	})

	t.Run("returns empty map when no headers are present", func(t *testing.T) {
		sections := extractSections("just a plain line\nand another one")
		if len(sections) != 0 {
			t.Errorf("expected no sections, got %v", sections)
		}
	})

	t.Run("header-only line yields empty section content", func(t *testing.T) {
		sections := extractSections("ROOT CAUSE:")
		got, ok := sections["ROOT CAUSE"]
		if !ok {
			t.Fatal("ROOT CAUSE section missing")
		}
		if strings.TrimSpace(got) != "" {
			t.Errorf("ROOT CAUSE = %q, want empty content", got)
		}
	})
}

func TestFormatAIResponse_Structured(t *testing.T) {
	disableColor(t)
	response := strings.Join([]string{
		"ERROR ANALYSIS: pods are crashing",
		"ROOT CAUSE: bad image tag",
		"STEPS TO RESOLVE:",
		"1. Check the deployment",
		"2. Update the image tag",
	}, "\n")

	out := formatAIResponse(response)

	for _, want := range []string{
		"ERROR ANALYSIS", "pods are crashing",
		"ROOT CAUSE", "bad image tag",
		"STEPS TO RESOLVE", "Check the deployment", "Update the image tag",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("formatAIResponse output missing %q\ngot:\n%s", want, out)
		}
	}
}

func TestFormatAIResponse_StepsWithoutNumbers(t *testing.T) {
	disableColor(t)
	// A STEPS section whose body has no "N." markers falls back to printing the
	// trimmed step text verbatim.
	response := strings.Join([]string{
		"ROOT CAUSE: misconfiguration",
		"STEPS TO RESOLVE:",
		"just do the needful",
	}, "\n")

	out := formatAIResponse(response)
	if !strings.Contains(out, "just do the needful") {
		t.Errorf("output missing verbatim step text\ngot:\n%s", out)
	}
}

func TestFormatAIResponse_FallbackNumberedSteps(t *testing.T) {
	disableColor(t)
	// No recognized headers, but numbered steps -> "SUGGESTED STEPS" fallback.
	response := "Here is what to do\n1. First step\n2. Second step"

	out := formatAIResponse(response)
	if !strings.Contains(out, "SUGGESTED STEPS") {
		t.Errorf("output missing SUGGESTED STEPS fallback\ngot:\n%s", out)
	}
	if !strings.Contains(out, "First step") || !strings.Contains(out, "Second step") {
		t.Errorf("output missing step text\ngot:\n%s", out)
	}
}

func TestFormatAIResponse_FallbackPlainText(t *testing.T) {
	disableColor(t)
	// No headers, no numbered steps -> "AI ANALYSIS" fallback echoing the text.
	response := "connection refused to the cluster, check your kubeconfig"

	out := formatAIResponse(response)
	if !strings.Contains(out, "AI ANALYSIS") {
		t.Errorf("output missing AI ANALYSIS fallback\ngot:\n%s", out)
	}
	if !strings.Contains(out, "connection refused") {
		t.Errorf("output missing original text\ngot:\n%s", out)
	}
}

func TestFormatFallbackResponse(t *testing.T) {
	disableColor(t)

	t.Run("numbered steps", func(t *testing.T) {
		out := formatFallbackResponse("1. alpha\n2. beta")
		if !strings.Contains(out, "SUGGESTED STEPS") {
			t.Errorf("want SUGGESTED STEPS, got:\n%s", out)
		}
		if !strings.Contains(out, "alpha") || !strings.Contains(out, "beta") {
			t.Errorf("want step text, got:\n%s", out)
		}
	})

	t.Run("plain text", func(t *testing.T) {
		out := formatFallbackResponse("nothing structured here")
		if !strings.Contains(out, "AI ANALYSIS") {
			t.Errorf("want AI ANALYSIS, got:\n%s", out)
		}
		if !strings.Contains(out, "nothing structured here") {
			t.Errorf("want original text, got:\n%s", out)
		}
	})
}
