package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/sashabaranov/go-openai"

	"github.com/clouddrove/smurf/internal/ci"
)

// defaultModel is used when OPENAI_MODEL is not set.
const defaultModel = openai.GPT4oMini

// aiRequestTimeout bounds how long a single OpenAI call may take.
const aiRequestTimeout = 15 * time.Second

// Cost controls. Without a ceiling a single unlucky error could bill an
// arbitrarily long completion, and the default sampling temperature is tuned
// for open-ended writing rather than for reproducing the same diagnosis for
// the same failure.
const (
	maxResponseTokens = 700
	responseTemp      = 0.2

	// maxErrorChars bounds what is sent upstream. Terraform and Helm failures
	// can run to tens of kilobytes of plan or manifest output, all of it
	// billed. The tail is kept rather than the head because the actual cause
	// is almost always at the end of such output.
	maxErrorChars = 6000
)

// modelFromEnv returns the model to use for chat completions, allowing an
// override via the OPENAI_MODEL environment variable and falling back to
// defaultModel otherwise.
func modelFromEnv() string {
	if model := os.Getenv("OPENAI_MODEL"); model != "" {
		return model
	}
	return defaultModel
}

// IsEnabled reports whether an AI call can be attempted, printing actionable
// guidance when it cannot. A local endpoint needs no key, so this asks the
// resolved provider rather than looking for one specific variable.
func IsEnabled() bool {
	missing, guidance := resolveProvider().credentialsMissing()
	if missing {
		color.Yellow("⚠️  " + guidance)
		return false
	}
	return true
}

// truncateError bounds the error text sent upstream, keeping the tail. The
// marker matters: without it the model silently reasons about a fragment it
// believes is whole.
func truncateError(errText string) string {
	if len(errText) <= maxErrorChars {
		return errText
	}
	return "[earlier output truncated]\n" + errText[len(errText)-maxErrorChars:]
}

// invokedCommand reports the smurf subcommand being run, for example
// "stf apply". The same message means different things depending on which tool
// produced it, and the advice should differ with it.
//
// This is read from os.Args rather than threaded through a parameter because
// AIExplainError has over a hundred call sites; changing that signature would
// be churn out of all proportion to the benefit.
func invokedCommand() string {
	args := os.Args
	if len(args) < 2 {
		return ""
	}
	var parts []string
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "-") {
			break
		}
		parts = append(parts, a)
		if len(parts) == 3 {
			break
		}
	}
	return strings.Join(parts, " ")
}

// commandContext renders the failing command for the prompt, or nothing when
// it cannot be determined.
func commandContext() string {
	cmd := invokedCommand()
	if cmd == "" {
		return ""
	}
	return "The user was running: smurf " + cmd + "\n"
}

// Explain error in human readable format using AI
func ExplainError(errText string) (string, error) {
	errText = truncateError(Redact(errText))
	prompt := fmt.Sprintf(`
You are a Senior DevOps Engineer AI.

Your job is to analyze the error and generate SHORT, CRISP, TECHNICAL troubleshooting steps.
Avoid long sentences. Use direct bullet-style instructions.

STRICT FORMAT:

ROOT CAUSE:
- 1 short line

STEPS TO RESOLVE:
Step 1: <Short Title>
  - <Very short actionable check>
  - <File/command to verify>
  - <What to confirm>

Step 2: <Short Title>
  - <Short bullet>
  - <Command or file>
  - <Confirm outcome>

Step 3: <Short Title>
  - <Short bullet>
  - <Command>
  - <Expected result>

STYLE RULES:
- No long paragraphs.
- No storytelling.
- Use crisp bullets.
- Use Kubernetes/Docker/CI/CD/Helm style commands.
- Every sub-step must be actionable.

%sError to analyze: %s
`, commandContext(), errText)

	response, err := AskAI(prompt)
	if err != nil {
		return "", err
	}

	// Format the response with colors
	formatted := formatAIResponse(response)
	return formatted, nil
}

// Generic AI call
func AskAI(prompt string) (string, error) {
	provider := resolveProvider()
	if missing, guidance := provider.credentialsMissing(); missing {
		return "", errors.New(guidance)
	}

	// Serving a repeated failure from cache is the difference between a broken
	// pipeline costing one call and costing one per re-run.
	if cached, ok := cacheLookup(provider, prompt); ok {
		return cached, nil
	}

	// NewClientWithConfig rather than NewClient so OPENAI_BASE_URL can point at
	// any OpenAI-compatible endpoint: a local Ollama server, a free hosted
	// tier, or a different vendor entirely.
	cfg := openai.DefaultConfig(provider.authToken())
	if provider.BaseURL != "" {
		cfg.BaseURL = provider.BaseURL
	}
	client := openai.NewClientWithConfig(cfg)

	// Build request
	req := openai.ChatCompletionRequest{
		Model: provider.Model,
		// MaxTokens rather than MaxCompletionTokens: the newer field is an
		// OpenAI addition for the o1 series, and the compatible servers this
		// now targets, Ollama and llama.cpp among them, understand only
		// max_tokens. Choosing the deprecated field keeps the cost ceiling
		// effective everywhere instead of only against OpenAI.
		MaxTokens:   maxResponseTokens, //nolint:staticcheck // SA1019: see above
		Temperature: responseTemp,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Call OpenAI API with a bounded timeout so a slow/unreachable API never
	// hangs the CLI indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), aiRequestTimeout)
	defer cancel()

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		// The endpoint is named because with a custom base URL "openai error"
		// would point at the wrong service entirely.
		return "", fmt.Errorf("%s error: %v", providerLabel(provider), err)
	}

	// Prevent panic – Always validate response
	if len(resp.Choices) == 0 {
		return "", errors.New("AI response is empty")
	}

	answer := resp.Choices[0].Message.Content

	// An empty completion is a failure, not an answer. Free tiers return one
	// under load, and rendering it produces an "AI ANALYSIS:" heading with
	// nothing beneath, which reads as a broken tool rather than an unavailable
	// service. It is also not worth caching.
	if strings.TrimSpace(answer) == "" {
		return "", errors.New("AI response is empty")
	}

	cacheStore(provider, prompt, answer)
	return answer, nil
}

// providerLabel names the endpoint in error messages.
func providerLabel(p providerConfig) string {
	if p.BaseURL == "" {
		return "openai"
	}
	return p.BaseURL
}

// formatAIResponse formats the AI response with colors
func formatAIResponse(response string) string {
	var output strings.Builder

	sections := extractSections(response)
	if len(sections) == 0 {
		return formatFallbackResponse(response)
	}

	renderErrorAnalysis(&output, sections)
	renderRootCause(&output, sections)
	renderSteps(&output, sections, response)

	return output.String()
}

func renderErrorAnalysis(output *strings.Builder, sections map[string]string) {
	red := color.New(color.FgRed, color.Bold)
	white := color.New(color.FgWhite)

	analysis, ok := sections["ERROR ANALYSIS"]
	if !ok {
		return
	}

	red.Fprint(output, "🚨 ERROR ANALYSIS\n")
	white.Fprint(output, strings.TrimSpace(analysis)+"\n\n")
}

func renderRootCause(output *strings.Builder, sections map[string]string) {
	yellow := color.New(color.FgYellow, color.Bold)
	white := color.New(color.FgWhite)

	cause, ok := sections["ROOT CAUSE"]
	if !ok {
		return
	}

	yellow.Fprint(output, "🔍 ROOT CAUSE\n")
	white.Fprint(output, strings.TrimSpace(cause)+"\n\n")
}

func renderSteps(
	output *strings.Builder,
	sections map[string]string,
	response string,
) {
	green := color.New(color.FgGreen, color.Bold)

	steps, ok := sections["STEPS TO RESOLVE"]
	if !ok {
		output.WriteString(formatFallbackResponse(response))
		return
	}

	green.Fprint(output, "📋 STEPS TO RESOLVE\n")
	printSteps(output, steps)
}

func printSteps(output *strings.Builder, steps string) {
	cyan := color.New(color.FgCyan, color.Bold)
	white := color.New(color.FgWhite)

	// The prompt asks for "Step 1: Title", so that is what this matches. It
	// previously looked for "1. Title", which the prompt never requests, so
	// this renderer never ran and every response fell through to plain text.
	// Numbered form is still accepted since models drift toward it.
	stepRegex := regexp.MustCompile(`(?m)^\s*(?:Step\s+)?(\d+)[.:]\s+(.+)$`)
	matches := stepRegex.FindAllStringSubmatch(steps, -1)

	if len(matches) == 0 {
		white.Fprint(output, strings.TrimSpace(steps)+"\n")
		return
	}

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		cyan.Fprint(output, match[1]+". ")
		white.Fprint(output, match[2]+"\n")
	}
}

// extractSections attempts to extract sections from the AI response
func extractSections(response string) map[string]string {
	sections := make(map[string]string)

	// Common section headers
	headers := []string{"ERROR ANALYSIS", "ROOT CAUSE", "STEPS TO RESOLVE", "SOLUTION", "STEPS"}

	currentSection := ""
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upperLine := strings.ToUpper(trimmed)

		// Check if line is a section header
		isHeader := false
		for _, header := range headers {
			if strings.HasPrefix(upperLine, header+":") {
				currentSection = header
				// Find the colon's index in the original trimmed string to correctly split.
				colonIndex := strings.Index(trimmed, ":")
				content := ""
				if colonIndex != -1 {
					content = strings.TrimSpace(trimmed[colonIndex+1:])
				}
				if content != "" {
					sections[currentSection] = content + "\n"
				} else {
					sections[currentSection] = ""
				}
				isHeader = true
				break
			}
		}

		if !isHeader && currentSection != "" {
			sections[currentSection] += trimmed + "\n"
		}
	}

	return sections
}

// formatFallbackResponse provides fallback formatting when sections aren't found
func formatFallbackResponse(response string) string {
	var output strings.Builder

	green := color.New(color.FgGreen, color.Bold)
	white := color.New(color.FgWhite)

	// Try to extract numbered steps
	stepRegex := regexp.MustCompile(`(\d+)\.\s+(.+)`)
	stepMatches := stepRegex.FindAllStringSubmatch(response, -1)

	if len(stepMatches) > 0 {
		green.Fprint(&output, "📋 SUGGESTED STEPS:\n")
		for _, match := range stepMatches {
			if len(match) >= 3 {
				green.Fprint(&output, match[1]+". ")
				white.Fprint(&output, match[2]+"\n")
			}
		}
	} else {
		green.Fprint(&output, "🤖 AI ANALYSIS:\n")
		white.Fprint(&output, response)
	}

	return output.String()
}

// The guard exists because a failure surfaces more than once: an internal
// function explains it, then the wrapper in cmd sees the same error return.
//
// It keys on the error text rather than latching a single boolean. A boolean
// was wrong: helm/rollback.go and helm/status.go both explain a tolerated
// sub-failure and then return nil, so in a chained command like selm provision
// that success path consumed the only slot and the later genuine failure was
// never explained at all.
var (
	explainedMu   sync.Mutex
	explainedSeen = map[string]bool{}
)

// resetExplainedForTest clears the record between tests.
func resetExplainedForTest() {
	explainedMu.Lock()
	defer explainedMu.Unlock()
	explainedSeen = map[string]bool{}
}

// alreadyExplained reports whether this exact error has been explained, and
// records it otherwise. Recording happens only for errors that are actually
// going to be explained, so a run with no key configured does not silently
// consume anything.
func alreadyExplained(errText string) bool {
	explainedMu.Lock()
	defer explainedMu.Unlock()

	// Containment, not equality. Go wraps errors as fmt.Errorf("...: %w", err),
	// so the same failure reaches this twice with different text: an internal
	// function explains "found 1 failed pods in release demo" and the wrapper
	// in cmd then sees "upgrade failed: found 1 failed pods in release demo".
	// Comparing exactly would print two analyses of one failure.
	for seen := range explainedSeen {
		if strings.Contains(errText, seen) || strings.Contains(seen, errText) {
			return true
		}
	}
	explainedSeen[errText] = true
	return false
}

func AIExplainError(useAI bool, errTest string) {
	if !useAI {
		return
	}
	if IsEnabled() {
		if alreadyExplained(errTest) {
			return
		}
		fmt.Println("\n🤖 Smurf AI Analysis...")
		answer, err := ExplainError(errTest)
		if err != nil {
			pterm.Error.Printf("AI analysis failed: %v\n", err)
			return
		}
		fmt.Println(answer)

		// The whole point of the analysis is telling someone what went wrong,
		// and in Actions stdout is the log wall the job summary exists to
		// avoid. Publishing it there puts the root cause at the top of the run
		// beside the pod table, rather than hundreds of lines down.
		publishAnalysisToCI(answer)
	}
}

// publishAnalysisToCI adds the explanation to the GitHub Actions job summary.
// It is a no-op elsewhere, and strips the ANSI colouring that makes sense in a
// terminal but renders as escape sequences in markdown.
func publishAnalysisToCI(answer string) {
	if !ci.SummaryEnabled() {
		return
	}
	plain := stripANSI(answer)
	if strings.TrimSpace(plain) == "" {
		return
	}
	ci.Summary("### 🤖 AI analysis\n\n" + ci.CodeBlock("text", plain))
}

// ansiPattern matches the escape sequences pterm and color emit.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes terminal colouring so the text reads as markdown.
func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
