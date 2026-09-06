package ai

import "regexp"

// redactionPatterns matches common secret shapes so they are never sent to the
// AI provider. The set is intentionally conservative to avoid mangling
// legitimate error text: it keys off recognisable token prefixes and explicit
// credential assignments rather than trying to spot high-entropy strings.
var redactionPatterns = []*regexp.Regexp{
	// GitHub fine-grained personal access tokens, e.g. github_pat_11ABC...
	// Listed before the gh*_ prefixes below purely for readability; the two
	// shapes cannot overlap, since "github_pat_" contains no gh?_ prefix.
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]+`),
	// GitHub tokens by prefix: personal access (ghp_), OAuth (gho_), server
	// to server (ghs_), user to server (ghu_) and refresh (ghr_).
	regexp.MustCompile(`gh[posur]_[A-Za-z0-9]+`),
	// AWS access key IDs, e.g. AKIAIOSFODNN7EXAMPLE
	regexp.MustCompile(`AKIA[A-Z0-9]{16}`),
	// Bearer tokens in Authorization headers. The class covers base64 and
	// base64url alphabets, so a JWT or a padded base64 token is caught whole
	// rather than truncated at its first + or /.
	regexp.MustCompile(`Bearer [A-Za-z0-9._+/=-]+`),
	// password=, token= and secret= style assignments. The quoted form is
	// listed first so the unquoted pattern below cannot stop at the first
	// space inside the quotes and leave the rest of the value exposed.
	regexp.MustCompile(`(?i)(password|token|secret)\s*=\s*"[^"]*"`),
	regexp.MustCompile(`(?i)(password|token|secret)\s*=\s*[^\s"]+`),
}

const redactedPlaceholder = "[REDACTED]"

// Redact masks substrings in s that look like common secrets (GitHub tokens,
// AWS access key IDs, Bearer tokens, and password=, token= or secret=
// assignments) before the text is sent to an external AI provider.
func Redact(s string) string {
	for _, re := range redactionPatterns {
		s = re.ReplaceAllString(s, redactedPlaceholder)
	}
	return s
}
