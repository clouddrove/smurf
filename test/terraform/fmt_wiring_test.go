package terraform_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clouddrove/smurf/cmd"
	_ "github.com/clouddrove/smurf/cmd/stf"
)

// findStf returns the stf command group.
func findStf(t *testing.T) *cobra.Command {
	t.Helper()
	for _, c := range cmd.RootCmd.Commands() {
		if c.Name() == "stf" {
			return c
		}
	}
	require.Fail(t, "stf is not registered on the root command")
	return nil
}

// TestFmtAcceptsFormatAlias covers a break that only showed up downstream.
//
// The command has always been `fmt`, but the shared Terraform workflow in
// clouddrove/github-shared-workflows called `smurf stf format`. While the stf
// group accepted an unknown subcommand as a positional argument, that printed a
// hint and exited 0, so the step passed for as long as it existed having
// formatted nothing. Adding Args: cobra.NoArgs to the group turned the same
// call into exit 1, and every caller tracking the latest release went red
// without changing anything of their own.
//
// The group guard is correct and stays. The alias is what lets those callers
// keep working while they move to `fmt`.
func TestFmtAcceptsFormatAlias(t *testing.T) {
	stf := findStf(t)

	found, _, err := stf.Find([]string{"format"})
	require.NoError(t, err, "smurf stf format must resolve to a command")
	assert.Equal(t, "fmt", found.Name(),
		"smurf stf format must dispatch to the fmt command, not fall through to the group")
}

// TestFmtHasDirFlag keeps fmt in line with the rest of the group. Every other
// stf subcommand takes --dir, and fmt not taking one meant it always ran
// against the process working directory: in a matrix build over several
// Terraform roots, one badly formatted file failed every root at once rather
// than the one at fault.
func TestFmtHasDirFlag(t *testing.T) {
	stf := findStf(t)

	found, _, err := stf.Find([]string{"fmt"})
	require.NoError(t, err)

	flag := found.Flags().Lookup("dir")
	require.NotNil(t, flag, "smurf stf fmt must accept --dir like every other stf subcommand")
	assert.Equal(t, ".", flag.DefValue, "--dir must default to the current directory")
}
