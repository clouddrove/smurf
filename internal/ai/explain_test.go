package ai

import (
	"os"
	"strings"
	"testing"
)

// promptShapedResponse is exactly what the prompt in ExplainError instructs the
// model to return. The renderer used to look for "1. Title" while the prompt
// asked for "Step 1: Title", so this shape never matched and every response
// fell through to unformatted text. Pinning the real shape here keeps the
// prompt and the renderer from drifting apart again.
const promptShapedResponse = `ROOT CAUSE:
- Image tag does not exist in the registry

STEPS TO RESOLVE:
Step 1: Verify the tag exists
  - Run docker images
  - Confirm the tag is present

Step 2: Re-tag the image
  - Run smurf sdkr tag app:dev app:prod
  - Confirm the output
`

func TestFormatter_RendersTheShapeThePromptAsksFor(t *testing.T) {
	out := formatAIResponse(promptShapedResponse)

	if !strings.Contains(out, "ROOT CAUSE") {
		t.Error("root cause section should be rendered")
	}
	if !strings.Contains(out, "STEPS TO RESOLVE") {
		t.Error("steps section should be rendered")
	}
	for _, want := range []string{"Verify the tag exists", "Re-tag the image"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain step title %q", want)
		}
	}
}

// Models drift toward plain numbering even when asked for "Step N:", so both
// are accepted rather than only the form the prompt requests.
func TestFormatter_AcceptsPlainNumberedSteps(t *testing.T) {
	response := `ROOT CAUSE:
- Bad credentials

STEPS TO RESOLVE:
1. Check the token
2. Re-authenticate
`
	out := formatAIResponse(response)

	for _, want := range []string{"Check the token", "Re-authenticate"} {
		if !strings.Contains(out, want) {
			t.Errorf("output should contain %q", want)
		}
	}
}

// Whatever the model returns, the text must reach the user. Dropping content
// because it did not match an expected shape would be worse than ugly output.
func TestFormatter_UnstructuredResponseStillReachesTheUser(t *testing.T) {
	response := "The registry rejected the push because the repository does not exist."

	out := formatAIResponse(response)

	if !strings.Contains(out, "repository does not exist") {
		t.Errorf("unstructured content must not be swallowed, got:\n%s", out)
	}
}

func TestTruncateError_ShortInputUnchanged(t *testing.T) {
	in := "something went wrong"
	if got := truncateError(in); got != in {
		t.Errorf("short input should pass through unchanged, got %q", got)
	}
}

// The cause of a Terraform or Helm failure is almost always at the end of a
// long output, so the tail is what must survive.
func TestTruncateError_KeepsTheTailAndSaysSo(t *testing.T) {
	in := strings.Repeat("noise\n", 5000) + "FINAL: the actual cause"

	got := truncateError(in)

	if len(got) > maxErrorChars+64 {
		t.Errorf("truncated length = %d, want roughly %d", len(got), maxErrorChars)
	}
	if !strings.Contains(got, "FINAL: the actual cause") {
		t.Error("the tail carries the cause and must be kept")
	}
	if !strings.Contains(got, "truncated") {
		t.Error("the model must be told the input was cut, or it reasons about a fragment as if it were whole")
	}
}

func TestInvokedCommand(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"group and subcommand": {[]string{"smurf", "stf", "apply"}, "stf apply"},
		"stops at flags":       {[]string{"smurf", "sdkr", "build", "--file", "Dockerfile"}, "sdkr build"},
		"top level only":       {[]string{"smurf", "version"}, "version"},
		"three levels":         {[]string{"smurf", "selm", "repo", "add", "name", "url"}, "selm repo add"},
		"no subcommand":        {[]string{"smurf"}, ""},
		"nothing at all":       {[]string{}, ""},
	}

	original := os.Args
	t.Cleanup(func() { os.Args = original })

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			os.Args = tc.args
			if got := invokedCommand(); got != tc.want {
				t.Errorf("invokedCommand() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The same message means different things from different tools, so the command
// belongs in the prompt.
func TestCommandContext(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"smurf", "stf", "apply"}
	got := commandContext()
	if !strings.Contains(got, "stf apply") {
		t.Errorf("commandContext() = %q, want it to name the command", got)
	}

	os.Args = []string{"smurf"}
	if got := commandContext(); got != "" {
		t.Errorf("commandContext() = %q, want empty when there is no command", got)
	}
}

// Redaction has to happen before truncation, or a secret near the start of a
// long error would be cut away rather than masked, and a shorter error
// containing the same secret would leak it.
func TestExplainError_RedactsBeforeSending(t *testing.T) {
	secret := "ghp_1234567890abcdefABCDEF1234567890"
	if !strings.Contains(Redact("token "+secret), "[REDACTED]") {
		t.Fatal("precondition: Redact should mask a GitHub token")
	}
	if strings.Contains(truncateError(Redact("token "+secret)), secret) {
		t.Error("the secret must not survive the redact-then-truncate path")
	}
}

// A failure surfaces more than once: an internal function explains it, then the
// wrapper in cmd sees the same error return. The same text should be explained
// once.
func TestAIExplainError_SameErrorExplainedOnce(t *testing.T) {
	resetExplainedForTest()

	if alreadyExplained("boom") {
		t.Fatal("a fresh error should not be marked as already explained")
	}
	if !alreadyExplained("boom") {
		t.Error("the same error text should be recognised the second time")
	}
}

// The reason this is keyed on the error rather than latched once per process:
// helm/rollback.go and helm/status.go both explain a tolerated sub-failure and
// then return nil. With a single latch, that success path consumed the only
// slot and a later genuine failure in the same run went unexplained.
func TestAIExplainError_DifferentErrorsAreBothExplained(t *testing.T) {
	resetExplainedForTest()

	if alreadyExplained("status retrieval failed") {
		t.Fatal("first error should be new")
	}
	if alreadyExplained("upgrade failed: timed out") {
		t.Error("a different, later failure must still be explained; it is the one that matters")
	}
}

// Without the flag nothing should happen, and in particular nothing should be
// recorded, or a later command that did ask would be skipped.
func TestAIExplainError_WithoutFlagRecordsNothing(t *testing.T) {
	resetExplainedForTest()

	AIExplainError(false, "a failure")

	if alreadyExplained("a failure") {
		t.Error("a call without --ai must not record the error as explained")
	}
}
