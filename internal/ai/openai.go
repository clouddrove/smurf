package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/sashabaranov/go-openai"
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
		Model:       provider.Model,
		MaxTokens:   maxResponseTokens,
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

func AIExplainError(useAI bool, errTest string) {
	if useAI && IsEnabled() {
		fmt.Println("\n🤖 Smurf AI Analysis...")
		answer, err := ExplainError(errTest)
		if err != nil {
			pterm.Error.Printf("AI analysis failed: %v\n", err)
			return
		}
		fmt.Println(answer)
	}
}
