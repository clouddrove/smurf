package sdkr

import (
	"strings"
	"testing"
)

// Regression coverage for validateGHCRImage per #387 review note #4.
// Pure function returning an error, so we just exercise each branch.
func TestValidateGHCRImage(t *testing.T) {
	cases := []struct {
		name    string
		image   string
		wantErr bool
		// wantSubstr is an optional message substring so we don't have
		// to lock in the exact human-formatted error text.
		wantSubstr string
	}{
		{
			name:    "canonical owner + repo + tag",
			image:   "ghcr.io/acme/example:1.0.0",
			wantErr: false,
		},
		{
			name:    "nested path owner + team + repo",
			image:   "ghcr.io/acme/team/example:1.0.0",
			wantErr: false,
		},
		{
			name:       "wrong registry prefix",
			image:      "docker.io/acme/example:1.0.0",
			wantErr:    true,
			wantSubstr: "invalid GHCR image format",
		},
		{
			name:       "missing owner",
			image:      "ghcr.io/example:1.0.0",
			wantErr:    true,
			wantSubstr: "missing owner",
		},
		{
			name:       "unparseable reference",
			image:      "ghcr.io/ACME_Invalid@@/example",
			wantErr:    true,
			wantSubstr: "invalid GHCR image reference",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGHCRImage(tc.image)
			if tc.wantErr && err == nil {
				t.Fatalf("expected an error for %q, got nil", tc.image)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.image, err)
			}
			if tc.wantErr && tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}
