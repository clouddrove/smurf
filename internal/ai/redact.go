package ai

import "regexp"

// redactionPatterns matches common secret shapes so they are never sent to the
// AI provider. The set is intentionally conservative (GitHub tokens, AWS access
// key IDs, Bearer tokens, and password=... assignments) to avoid mangling
// legitimate error text.
var redactionPatterns = []*regexp.Regexp{
	// GitHub personal access tokens, e.g. ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	regexp.MustCompile(`ghp_[A-Za-z0-9]+`),
	// AWS access key IDs, e.g. AKIAIOSFODNN7EXAMPLE
	regexp.MustCompile(`AKIA[A-Z0-9]{16}`),
	// Bearer tokens in Authorization headers
	regexp.MustCompile(`Bearer [A-Za-z0-9._-]+`),
	// password=... style assignments, quoted value first so an unquoted match
	// below doesn't stop at the first space inside the quotes.
	regexp.MustCompile(`(?i)password\s*=\s*"[^"]*"`),
	regexp.MustCompile(`(?i)password\s*=\s*[^\s"]+`),
}

const redactedPlaceholder = "[REDACTED]"

// Redact masks substrings in s that look like common secrets (GitHub tokens,
// AWS access key IDs, Bearer tokens, password=... values) before the text is
// sent to an external AI provider.
func Redact(s string) string {
	for _, re := range redactionPatterns {
		s = re.ReplaceAllString(s, redactedPlaceholder)
	}
	return s
}
