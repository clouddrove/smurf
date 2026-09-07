package ci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsGitHubActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	if !IsGitHubActions() {
		t.Error("should detect Actions when GITHUB_ACTIONS=true")
	}
	// The runner sets exactly "true"; anything else is not a runner.
	for _, v := range []string{"", "false", "1", "TRUE"} {
		t.Setenv("GITHUB_ACTIONS", v)
		if IsGitHubActions() {
			t.Errorf("GITHUB_ACTIONS=%q should not count as Actions", v)
		}
	}
}

// A raw newline ends a workflow command, so an unescaped multi-line message
// loses everything after the first line. That is the failure this guards.
func TestEscapeMessage(t *testing.T) {
	cases := map[string]string{
		"plain":            "plain",
		"line one\nline 2": "line one%0Aline 2",
		"has\rcarriage":    "has%0Dcarriage",
		"100% failed":      "100%25 failed",
		// Percent must be escaped before the escapes it would otherwise corrupt.
		"%0A literal": "%250A literal",
	}
	for in, want := range cases {
		if got := escapeMessage(in); got != want {
			t.Errorf("escapeMessage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEscapeProperty(t *testing.T) {
	got := escapeProperty("pod: api, ns: prod")
	for _, bad := range []string{":", ","} {
		if strings.Contains(got, bad) {
			t.Errorf("escapeProperty left %q unescaped: %s", bad, got)
		}
	}
}

type stringWriter struct{ strings.Builder }

func TestAnnotateTo(t *testing.T) {
	var w stringWriter
	annotateTo(&w, LevelError, "Helm upgrade failed", "pod api-7d9f: ImagePullBackOff")

	got := w.String()
	if !strings.HasPrefix(got, "::error title=Helm upgrade failed::") {
		t.Errorf("annotation = %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Error("annotation must end with a newline or the next output joins it")
	}
}

func TestAnnotateTo_NoTitle(t *testing.T) {
	var w stringWriter
	annotateTo(&w, LevelWarning, "", "something")

	if got := w.String(); !strings.HasPrefix(got, "::warning::") {
		t.Errorf("annotation = %q, want no title property", got)
	}
}

// Outside Actions this must be silent, so command code can call it
// unconditionally without a developer at a terminal seeing markup.
func TestAnnotate_SilentOutsideActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	// Annotate writes to os.Stdout; the assertion here is simply that it does
	// not panic and takes the early return. Content is covered by annotateTo.
	Annotate(LevelError, "title", "message")
}

func TestSummary_AppendsToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", path)

	Summary("## first")
	Summary("## second")

	data, err := os.ReadFile(path) // #nosec G304 -- path is this test's own t.TempDir file
	if err != nil {
		t.Fatalf("summary not written: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "## first") || !strings.Contains(got, "## second") {
		t.Errorf("summary should accumulate both writes, got:\n%s", got)
	}
	// Without a trailing newline the next append would join the previous line
	// and break the markdown.
	if !strings.Contains(got, "## first\n") {
		t.Error("each write should end with a newline")
	}
}

func TestSummary_NoDestinationIsSafe(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	Summary("## ignored") // must not panic
	if SummaryEnabled() {
		t.Error("SummaryEnabled should be false with no destination")
	}
}

// A summary that cannot be written must not fail a deploy that otherwise
// succeeded.
func TestSummary_UnwritablePathIsIgnored(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", filepath.Join(t.TempDir(), "nope", "summary.md"))
	Summary("## ignored") // parent directory does not exist; must not panic
}

func TestTable(t *testing.T) {
	got := Table([]string{"Pod", "Reason"}, [][]string{
		{"api-7d9f", "ImagePullBackOff"},
		{"api-7d9g", "CrashLoopBackOff"},
	})

	for _, want := range []string{"| Pod | Reason |", "|---|---|", "api-7d9f", "CrashLoopBackOff"} {
		if !strings.Contains(got, want) {
			t.Errorf("table missing %q:\n%s", want, got)
		}
	}
}

func TestTable_EmptyRowsRenderNothing(t *testing.T) {
	if got := Table([]string{"Pod"}, nil); got != "" {
		t.Errorf("a header with no rows should render nothing, got %q", got)
	}
	if got := Table(nil, [][]string{{"x"}}); got != "" {
		t.Errorf("rows with no header should render nothing, got %q", got)
	}
}

// A pipe or newline in a Kubernetes message would otherwise split the row and
// shift every later column.
func TestTable_SanitisesCellContent(t *testing.T) {
	got := Table([]string{"Pod", "Reason"}, [][]string{
		{"api", "failed | badly\nsecond line"},
	})

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Errorf("expected header, separator and one row; got %d lines:\n%s", len(lines), got)
	}
	if strings.Contains(lines[2], "| badly") && !strings.Contains(lines[2], "\\| badly") {
		t.Errorf("pipe should be escaped: %s", lines[2])
	}
}

// A short row must not shift the columns of the rest of the table.
func TestTable_PadsShortRows(t *testing.T) {
	got := Table([]string{"A", "B", "C"}, [][]string{{"only-one"}})

	row := strings.Split(strings.TrimSpace(got), "\n")[2]
	if strings.Count(row, "|") != 4 {
		t.Errorf("row should have the same column count as the header: %q", row)
	}
}

func TestCodeBlock(t *testing.T) {
	got := CodeBlock("text", "line one\nline two\n")
	if !strings.HasPrefix(got, "```text\n") || !strings.HasSuffix(got, "```\n") {
		t.Errorf("code block malformed:\n%s", got)
	}
}

// The job summary has a size limit, and a job that fails to publish its summary
// reports nothing at all, so long output is cut rather than risking the whole
// thing.
func TestCodeBlock_TruncatesLongBodies(t *testing.T) {
	got := CodeBlock("", strings.Repeat("x", 20000))

	if len(got) > 9000 {
		t.Errorf("code block length = %d, want it truncated", len(got))
	}
	if !strings.Contains(got, "truncated") {
		t.Error("truncation should be visible to the reader")
	}
}
