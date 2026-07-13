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
