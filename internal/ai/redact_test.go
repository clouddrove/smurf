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

// assertRedacted checks that every secret is gone and a placeholder is left.
func assertRedacted(t *testing.T, input string, secrets ...string) {
	t.Helper()
	got := Redact(input)
	for _, secret := range secrets {
		if strings.Contains(got, secret) {
			t.Errorf("Redact(%q) = %q, still contains secret %q", input, got, secret)
		}
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("Redact(%q) = %q, want it to contain [REDACTED]", input, got)
	}
}

// TestRedact_GitHubTokenPrefixes covers the token types GitHub issues beyond
// the classic ghp_: OAuth, server-to-server, user-to-server, refresh, and the
// fine-grained PAT, whose body also contains underscores.
func TestRedact_GitHubTokenPrefixes(t *testing.T) {
	tokens := []string{
		"ghp_1234567890abcdefABCDEF1234567890",
		"gho_16C7e42F292c6912E7710c838347Ae178B4a",
		"ghs_1234567890abcdefABCDEF1234567890",
		"ghu_1234567890abcdefABCDEF1234567890",
		"ghr_1234567890abcdefABCDEF1234567890",
		"github_pat_11ABCDEFG0abcdefGHIJKL_mnopQRSTUV12345",
	}

	for _, token := range tokens {
		t.Run(token[:strings.Index(token, "_")+1], func(t *testing.T) {
			assertRedacted(t, "authentication with "+token+" failed", token)
		})
	}
}

// TestRedact_CredentialAssignments covers password=, token= and secret=, in
// both quoted and unquoted form. The quoted case matters most: the value holds
// spaces, so a pattern that stopped at the first space would leave the rest of
// the secret in the text sent upstream.
func TestRedact_CredentialAssignments(t *testing.T) {
	for _, key := range []string{"password", "token", "secret", "PASSWORD", "Token", "SECRET"} {
		t.Run(key+" unquoted", func(t *testing.T) {
			assertRedacted(t, "operation failed: "+key+"=hunter2hunter2 rejected", "hunter2hunter2")
		})
		t.Run(key+" quoted", func(t *testing.T) {
			assertRedacted(t, `config error: `+key+`="a b c value" is invalid`, "a b c value")
		})
	}
}

// TestRedact_BearerTokens covers base64 and base64url payloads. The character
// class previously stopped at the first + or /, so the tail of a token was
// sent upstream in the clear.
func TestRedact_BearerTokens(t *testing.T) {
	for name, tc := range map[string]struct {
		input   string
		secrets []string
	}{
		"base64 with plus slash and padding": {
			"Authorization: Bearer aGVsbG8+d29ybGQ/dGhpcw==",
			[]string{"aGVsbG8", "d29ybGQ", "dGhpcw"},
		},
		"jwt": {
			"rejected Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.aBc-_123",
			[]string{"eyJhbGciOiJIUzI1NiJ9", "aBc-_123"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertRedacted(t, tc.input, tc.secrets...)
		})
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
