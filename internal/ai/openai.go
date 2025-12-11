package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
)

// Check if API key exists or not
func IsEnabled() bool {
	if os.Getenv("OPENAI_API_KEY") == "" {
		color.Yellow("⚠️  AI mode enabled but no OPENAI_API_KEY found.")
		color.Cyan("👉 Run: export OPENAI_API_KEY=your_key_here")
		return false
	}
	return true
}

// Explain error in human readable format using AI
func ExplainError(errText string) (string, error) {
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

Error to analyze: %s
`, errText)

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
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY is not set")
	}

	// Create OpenAI Client
	client := openai.NewClient(apiKey)

	// Build request
	req := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini, // or openai.GPT3Dot5Turbo
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Call OpenAI API
	resp, err := client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("openai error: %v", err)
	}

	// Prevent panic – Always validate response
	if len(resp.Choices) == 0 {
		return "", errors.New("AI response is empty")
	}

	return resp.Choices[0].Message.Content, nil
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

	stepRegex := regexp.MustCompile(`(\d+)\.\s+(.+)`)
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
