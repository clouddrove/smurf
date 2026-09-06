package ai

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantAbsent  []string
		wantPresent string
	}{
		{
			name:        "github token",
			input:       "failed to push: authentication with ghp_1234567890abcdefABCDEF1234567890 failed",
			wantAbsent:  []string{"ghp_1234567890abcdefABCDEF1234567890"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "aws access key",
			input:       "invalid credentials for AKIAIOSFODNN7EXAMPLE in region us-east-1",
			wantAbsent:  []string{"AKIAIOSFODNN7EXAMPLE"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "bearer token",
			input:       "request failed with header Authorization: Bearer abc123.def456-ghi789",
			wantAbsent:  []string{"Bearer abc123.def456-ghi789"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "password assignment unquoted",
			input:       "connection string db://user:pass@host?password=SuperSecret123 failed",
			wantAbsent:  []string{"SuperSecret123"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "password assignment quoted",
			input:       `config error: password="super secret value" is invalid`,
			wantAbsent:  []string{"super secret value"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "github oauth token",
			input:       "auth failed for gho_16C7e42F292c6912E7710c838347Ae178B4a",
			wantAbsent:  []string{"gho_16C7e42F292c6912E7710c838347Ae178B4a"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "github server to server token",
			input:       "installation token ghs_1234567890abcdefABCDEF1234567890 rejected",
			wantAbsent:  []string{"ghs_1234567890abcdefABCDEF1234567890"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "github user to server token",
			input:       "ghu_1234567890abcdefABCDEF1234567890 is expired",
			wantAbsent:  []string{"ghu_1234567890abcdefABCDEF1234567890"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "github refresh token",
			input:       "refresh ghr_1234567890abcdefABCDEF1234567890 failed",
			wantAbsent:  []string{"ghr_1234567890abcdefABCDEF1234567890"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "github fine grained pat",
			input:       "denied for github_pat_11ABCDEFG0abcdefGHIJKL_mnopQRSTUV12345 on repo",
			wantAbsent:  []string{"github_pat_11ABCDEFG0abcdefGHIJKL_mnopQRSTUV12345"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "token assignment unquoted",
			input:       "registry login failed: token=abc123DEF456ghi789 rejected",
			wantAbsent:  []string{"abc123DEF456ghi789"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "token assignment quoted",
			input:       `helm error: token="some secret token" not accepted`,
			wantAbsent:  []string{"some secret token"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "secret assignment unquoted",
			input:       "terraform var secret=hunter2hunter2 is invalid",
			wantAbsent:  []string{"hunter2hunter2"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "secret assignment quoted",
			input:       `error: secret="a b c value" could not be read`,
			wantAbsent:  []string{"a b c value"},
			wantPresent: "[REDACTED]",
		},
		{
			// A base64 token contains + and / and ends in = padding. The old
			// Bearer class stopped at the first + or /, leaving the tail of
			// the token in the text sent upstream.
			name:        "bearer token with base64 characters",
			input:       "Authorization: Bearer aGVsbG8+d29ybGQ/dGhpcw==",
			wantAbsent:  []string{"aGVsbG8", "d29ybGQ", "dGhpcw"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "bearer jwt",
			input:       "rejected Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.aBc-_123",
			wantAbsent:  []string{"eyJhbGciOiJIUzI1NiJ9", "aBc-_123"},
			wantPresent: "[REDACTED]",
		},
		{
			name:        "case insensitive assignment keys",
			input:       "Error: Token=SHOUTYSECRET1 and SECRET=quietsecret2 rejected",
			wantAbsent:  []string{"SHOUTYSECRET1", "quietsecret2"},
			wantPresent: "[REDACTED]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact(tc.input)
			for _, absent := range tc.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("Redact(%q) = %q, still contains secret %q", tc.input, got, absent)
				}
			}
			if !strings.Contains(got, tc.wantPresent) {
				t.Errorf("Redact(%q) = %q, want it to contain %q", tc.input, got, tc.wantPresent)
			}
		})
	}
}

func TestRedact_LeavesNormalTextAlone(t *testing.T) {
	input := "Error: connection refused to api.example.com, retry after checking your kubeconfig"
	got := Redact(input)
	if got != input {
		t.Errorf("Redact(%q) = %q, want unchanged text", input, got)
	}
}

// TestRedact_LeavesLegitimateTextAlone guards the other direction. Broadening
// the patterns to token= and secret= risks eating ordinary error text, so
// these strings must survive untouched: they mention the keywords, or look
// token-shaped, without being credential assignments.
func TestRedact_LeavesLegitimateTextAlone(t *testing.T) {
	inputs := []string{
		"Error: the secret my-tls-cert was not found in namespace default",
		"failed to create token review for service account default",
		"github_actions runner is offline",
		"error parsing password policy document",
		"ghost_writer_test failed",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if got := Redact(input); got != input {
				t.Errorf("Redact(%q) = %q, want unchanged text", input, got)
			}
		})
	}
}

// TestRedact_BearerOverRedactsByDesign pins a known false positive. The Bearer
// pattern takes everything up to whitespace, so the prose "Bearer
// authentication" is masked too. That predates the widening of the character
// class in this change and is left as is deliberately: for a redactor, masking
// a word that was not a secret costs a little context in the AI prompt, while
// missing one leaks a credential to a third party. Failing closed is the right
// trade. This test exists so the behaviour is a decision, not a surprise.
func TestRedact_BearerOverRedactsByDesign(t *testing.T) {
	got := Redact("Bearer authentication is required but no header was supplied")
	want := "[REDACTED] is required but no header was supplied"
	if got != want {
		t.Errorf("Redact() = %q, want %q", got, want)
	}
}
