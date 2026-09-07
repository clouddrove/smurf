// Package ci renders output that GitHub Actions understands.
//
// smurf already knows a great deal when something fails: which pod is in
// ImagePullBackOff, which container exited non-zero, what the last event said.
// In a terminal pterm presents that well. In Actions the same output becomes a
// flat wall of log with escape codes in it, and the one line that matters is
// four hundred lines up.
//
// Two Actions features fix that, and neither needs a token or an API:
//
//   - workflow commands (::error::) attach a message to the run, so it appears
//     in the UI and in the pull request rather than only in the log
//   - the job summary is markdown rendered at the top of the run, which is the
//     first thing anyone opening a failed job actually reads
//
// Everything here is a no-op outside Actions, so command code can call it
// unconditionally and a developer at a terminal sees no change.
package ci

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Level maps to the GitHub workflow command of the same name.
type Level string

const (
	LevelError   Level = "error"
	LevelWarning Level = "warning"
	LevelNotice  Level = "notice"
)

// IsGitHubActions reports whether the process is running inside a GitHub
// Actions job. The runner sets GITHUB_ACTIONS=true for every step.
func IsGitHubActions() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}

// escapeMessage encodes a workflow command message.
//
// A raw newline would end the command and leave the rest of the text as
// ordinary log output, so multi-line detail silently loses everything after
// the first line. Percent must be escaped first, otherwise it would corrupt
// the escapes introduced afterwards.
func escapeMessage(s string) string {
	s = strings.ReplaceAll(s, "%", "%25")
	s = strings.ReplaceAll(s, "\r", "%0D")
	s = strings.ReplaceAll(s, "\n", "%0A")
	return s
}

// escapeProperty encodes an annotation property value, which additionally
// treats colon and comma as separators.
func escapeProperty(s string) string {
	s = escapeMessage(s)
	s = strings.ReplaceAll(s, ":", "%3A")
	s = strings.ReplaceAll(s, ",", "%2C")
	return s
}

// Annotate writes a workflow command so the message surfaces in the Actions UI.
// Outside Actions it does nothing, so callers need no conditional.
func Annotate(level Level, title, message string) {
	if !IsGitHubActions() {
		return
	}
	annotateTo(os.Stdout, level, title, message)
}

// annotateTo is the testable core of Annotate.
func annotateTo(w interface{ WriteString(string) (int, error) }, level Level, title, message string) {
	var props string
	if title != "" {
		props = " title=" + escapeProperty(title)
	}
	_, _ = w.WriteString(fmt.Sprintf("::%s%s::%s\n", level, props, escapeMessage(message)))
}

// Summary appends markdown to the job summary, which Actions renders at the
// top of the run. Failures are ignored on purpose: a summary that cannot be
// written is not a reason to fail a deploy that otherwise succeeded.
func Summary(markdown string) {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return
	}
	// The runner always provides an absolute path. Refusing anything else keeps
	// a stray relative value from writing into the working directory, which for
	// smurf could mean a chart or Terraform directory.
	if !filepath.IsAbs(path) {
		return
	}
	// 0600 because the summary carries error output, which can include paths
	// and cluster detail. The runner normally creates this file itself, so the
	// mode applies only when smurf creates it first.
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) // #nosec G304 G703 -- runner-provided absolute path, validated above
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	body := markdown
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	_, _ = f.WriteString(body)
}

// SummaryEnabled reports whether a job summary destination exists. Callers use
// it to skip assembling markdown that would be discarded.
func SummaryEnabled() bool {
	return os.Getenv("GITHUB_STEP_SUMMARY") != ""
}

// Table renders a markdown table, or an empty string when there are no rows so
// callers do not emit a header with nothing under it.
//
// Cell contents are sanitised: a stray pipe from a Kubernetes message would
// otherwise split the row and shift every later column, which is exactly the
// kind of quiet corruption that makes a summary untrustworthy.
func Table(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}
	var b strings.Builder

	writeRow := func(cells []string) {
		b.WriteString("|")
		for _, c := range cells {
			b.WriteString(" " + sanitizeCell(c) + " |")
		}
		b.WriteString("\n")
	}

	writeRow(headers)
	b.WriteString("|")
	for range headers {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	for _, row := range rows {
		// Pad or trim so a short row cannot silently shift columns.
		cells := make([]string, len(headers))
		copy(cells, row)
		writeRow(cells)
	}
	return b.String()
}

// sanitizeCell makes arbitrary text safe inside a markdown table cell.
func sanitizeCell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// CodeBlock wraps text in a fenced block for the job summary. Long output is
// truncated because the summary has a size limit and a job that fails to
// publish its summary reports nothing at all.
func CodeBlock(lang, body string) string {
	const maxBody = 8000
	if len(body) > maxBody {
		body = body[:maxBody] + "\n... output truncated ..."
	}
	return "```" + lang + "\n" + strings.TrimRight(body, "\n") + "\n```\n"
}
