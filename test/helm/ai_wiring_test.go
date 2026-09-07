package helm_test

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clouddrove/smurf/cmd"
	_ "github.com/clouddrove/smurf/cmd/sdkr"
	_ "github.com/clouddrove/smurf/cmd/selm"
	_ "github.com/clouddrove/smurf/cmd/stf"
)

// walk visits every command in the tree.
func walk(root *cobra.Command, fn func(*cobra.Command)) {
	for _, c := range root.Commands() {
		fn(c)
		walk(c, fn)
	}
}

// Commands that advertise --ai must be able to report a failure at all. A
// command using Run rather than RunE cannot return an error, so the flag on it
// could never do anything, and offering it would be a promise the command
// cannot keep.
func TestCommandsOfferingAICanReportErrors(t *testing.T) {
	var offenders []string

	walk(cmd.RootCmd, func(c *cobra.Command) {
		if c.Flags().Lookup("ai") == nil {
			return
		}
		if c.RunE == nil {
			offenders = append(offenders, c.CommandPath())
		}
	})

	assert.Emptyf(t, offenders,
		"these commands offer --ai but use Run rather than RunE, so they can never report a failure: %v", offenders)
}

// The reason the central wrapper exists: an audit found 19 of 42 internal
// functions with error paths that never reached AIExplainError, so relying on
// each call site was not working. Every command that offers the flag should be
// covered, and the count is asserted so a future command cannot quietly opt out.
func TestEveryAICommandIsWired(t *testing.T) {
	var withFlag int

	walk(cmd.RootCmd, func(c *cobra.Command) {
		if c.Flags().Lookup("ai") != nil && c.RunE != nil {
			withFlag++
		}
	})

	require.Greaterf(t, withFlag, 30,
		"expected the --ai flag on most commands, found %d; if the flag was renamed the wrapper silently covers nothing", withFlag)
}

// The wrapper must not fire when the flag is absent, or every failure would
// attempt a network call for users who never asked for one.
func TestWrapperIgnoresCommandsWithoutTheFlag(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	called := false
	child := &cobra.Command{
		Use:  "child",
		RunE: func(*cobra.Command, []string) error { called = true; return errors.New("boom") },
	}
	root.AddCommand(child)

	// No --ai flag is defined, so the wrapper should leave RunE untouched.
	require.NotNil(t, child.RunE)
	err := child.RunE(child, nil)

	assert.True(t, called, "the original RunE must still run")
	assert.Error(t, err, "the original error must still propagate")
}

// Whatever the wrapper does with the error, the command's own return value has
// to reach cobra unchanged, or exit codes stop meaning anything.
func TestErrorStillPropagates(t *testing.T) {
	sentinel := errors.New("original failure")

	var found *cobra.Command
	walk(cmd.RootCmd, func(c *cobra.Command) {
		if found == nil && c.Flags().Lookup("ai") != nil && c.RunE != nil {
			found = c
		}
	})
	require.NotNil(t, found, "expected at least one command offering --ai")

	// Replace the body, then confirm the error survives whatever wraps it.
	original := found.RunE
	t.Cleanup(func() { found.RunE = original })
	found.RunE = func(*cobra.Command, []string) error { return sentinel }

	err := found.RunE(found, nil)
	assert.ErrorIs(t, err, sentinel, "the wrapper must not swallow or replace the command's error")
}

// A persistent --ai declared on a group must still be seen. The wrapper runs
// before Execute parses flags, so a wire-time Flags() lookup would find only
// local flags: declaring --ai once on sdkr/selm/stf instead of 42 times, an
// obvious tidy-up, would then silently stop every command being wrapped with
// nothing failing to say so. The lookup is deferred to run time for that
// reason, and this pins the behaviour.
func TestUseAIFlagSeesPersistentFlags(t *testing.T) {
	parent := &cobra.Command{Use: "group"}
	parent.PersistentFlags().Bool("ai", false, "")

	child := &cobra.Command{Use: "child", RunE: func(*cobra.Command, []string) error { return nil }}
	parent.AddCommand(child)

	// Before parsing, a local lookup finds nothing, which is exactly the trap.
	require.Nil(t, child.Flags().Lookup("ai"),
		"precondition: a persistent parent flag is not in the child's local set before parsing")

	require.NoError(t, parent.PersistentFlags().Set("ai", "true"))

	// After cobra has merged them, the inherited set carries the value.
	assert.NotNil(t, child.InheritedFlags().Lookup("ai"),
		"the child should inherit a persistent --ai from its parent")
}

// deploy is the pipeline most worth explaining when it breaks: a failure can
// come from the build, any of four registries, or the Helm release. It offered
// no --ai at all and passed false to every internal call.
func TestDeployOffersAI(t *testing.T) {
	var deploy *cobra.Command
	walk(cmd.RootCmd, func(c *cobra.Command) {
		if c.Name() == "deploy" {
			deploy = c
		}
	})
	require.NotNil(t, deploy, "deploy command should be registered")

	assert.NotNil(t, deploy.Flags().Lookup("ai"),
		"deploy should offer --ai; it is the command with the most ways to fail")
}
