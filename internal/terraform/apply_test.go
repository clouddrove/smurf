package terraform

import (
	"errors"
	"testing"
)

// TestConfirmApproval verifies the approval gate shared by apply and destroy:
// a pre-approved run or an explicit "yes" proceeds (nil), while a declined
// prompt returns ErrOperationCancelled so the CLI exits non-zero.
func TestConfirmApproval(t *testing.T) {
	cases := []struct {
		name    string
		approve bool
		ask     func() bool
		wantErr error
	}{
		{
			name:    "auto-approve skips the prompt",
			approve: true,
			ask:     func() bool { t.Helper(); t.Error("ask must not be called when pre-approved"); return false },
			wantErr: nil,
		},
		{
			name:    "user approves",
			approve: false,
			ask:     func() bool { return true },
			wantErr: nil,
		},
		{
			name:    "user declines",
			approve: false,
			ask:     func() bool { return false },
			wantErr: ErrOperationCancelled,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := confirmApproval(tc.approve, tc.ask)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("confirmApproval(%v) error = %v, want %v", tc.approve, err, tc.wantErr)
			}
		})
	}
}
