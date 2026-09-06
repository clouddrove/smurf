package helm_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clouddrove/smurf/cmd"
	_ "github.com/clouddrove/smurf/cmd/sdkr"
	_ "github.com/clouddrove/smurf/cmd/selm"
	_ "github.com/clouddrove/smurf/cmd/stf"
)

// findGroup returns one of the three top level command groups.
func findGroup(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, c := range cmd.RootCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	require.Failf(t, "group not found", "%s is not registered on the root command", name)
	return nil
}

// TestGroupsRejectUnknownSubcommands guards a silent-success bug. sdkr, selm
// and stf each define Run with no Args validation, which made cobra treat an
// unknown subcommand as a positional argument: it ran the group's Run, printed
// a usage hint and exited 0.
//
// That is dangerous in a pipeline rather than merely untidy. `smurf stf aply`
// reported success while applying nothing, and the surrounding job carried on
// as though Terraform had run.
func TestGroupsRejectUnknownSubcommands(t *testing.T) {
	for _, group := range []string{"sdkr", "selm", "stf"} {
		t.Run(group, func(t *testing.T) {
			g := findGroup(t, group)

			require.NotNil(t, g.Args,
				"%s must declare Args; without it an unknown subcommand is accepted as a positional argument and exits 0", group)

			err := g.Args(g, []string{"totally-bogus-subcommand"})
			assert.Error(t, err,
				"smurf %s totally-bogus-subcommand must be rejected, not treated as a positional argument", group)
		})
	}
}

// TestGroupsStillAcceptTheirSubcommands is the other half. Args validation must
// reject stray arguments without breaking dispatch to real subcommands, so a
// fix for the above cannot quietly disable the commands themselves.
func TestGroupsStillAcceptTheirSubcommands(t *testing.T) {
	expected := map[string][]string{
		"sdkr": {"build", "scan", "tag", "remove", "push"},
		"selm": {"install", "upgrade", "rollback", "lint", "template", "list"},
		"stf":  {"init", "plan", "apply", "destroy", "validate", "fmt"},
	}

	for group, subs := range expected {
		t.Run(group, func(t *testing.T) {
			g := findGroup(t, group)

			registered := map[string]bool{}
			for _, c := range g.Commands() {
				registered[c.Name()] = true
			}

			for _, sub := range subs {
				assert.Truef(t, registered[sub], "smurf %s %s should be registered", group, sub)
			}

			// No args reaches Args when a subcommand matches, so this is what
			// dispatch actually relies on.
			assert.NoError(t, g.Args(g, []string{}),
				"%s must still accept being run with no arguments", group)
		})
	}
}
