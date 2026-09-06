package helm_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clouddrove/smurf/cmd"
	_ "github.com/clouddrove/smurf/cmd/selm"
)

// findSelmSubcommand returns the named subcommand of `smurf selm`.
func findSelmSubcommand(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, group := range cmd.RootCmd.Commands() {
		if group.Name() != "selm" {
			continue
		}
		for _, sub := range group.Commands() {
			if sub.Name() == name {
				return sub
			}
		}
	}
	require.Failf(t, "subcommand not found", "selm %s is not registered", name)
	return nil
}

// TestSelmTimeoutDefaultsAreIndependent guards against a regression where
// install, rollback and upgrade each bound --timeout to the same package-level
// variable. pflag writes a flag's default into the bound variable when the
// flag is registered, so the last init() to run decided the default for all
// three: every command silently used 120s no matter what its help text said.
//
// Value.String() is deliberate. It reports what the bound variable currently
// holds, which is the value a command actually uses when the user passes no
// --timeout. DefValue would not catch the bug, since pflag records that string
// per flag and it stayed correct even while the shared variable was wrong.
func TestSelmTimeoutDefaultsAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		command string
		want    string
	}{
		{"install", "600"},
		{"rollback", "300"},
		{"upgrade", "120"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			sub := findSelmSubcommand(t, tc.command)

			flag := sub.Flags().Lookup("timeout")
			require.NotNil(t, flag, "selm %s has no --timeout flag", tc.command)

			assert.Equal(t, tc.want, flag.Value.String(),
				"selm %s should default to %ss; a different value means --timeout is bound to a variable shared with another command",
				tc.command, tc.want)

			assert.Equal(t, tc.want, flag.DefValue,
				"selm %s help text should advertise %ss", tc.command, tc.want)
		})
	}
}

// TestSelmTimeoutFlagsAreDistinctVariables sets each command's --timeout and
// checks the others are unmoved. Independent defaults alone would not catch a
// future re-share, since three commands could coincidentally agree at init.
func TestSelmTimeoutFlagsAreDistinctVariables(t *testing.T) {
	install := findSelmSubcommand(t, "install")
	rollback := findSelmSubcommand(t, "rollback")
	upgrade := findSelmSubcommand(t, "upgrade")

	require.NoError(t, install.Flags().Set("timeout", "999"))

	assert.Equal(t, "999", install.Flags().Lookup("timeout").Value.String())
	assert.Equal(t, "300", rollback.Flags().Lookup("timeout").Value.String(),
		"setting install --timeout must not move rollback's")
	assert.Equal(t, "120", upgrade.Flags().Lookup("timeout").Value.String(),
		"setting install --timeout must not move upgrade's")

	// Restore so ordering between tests cannot matter.
	require.NoError(t, install.Flags().Set("timeout", "600"))
}
