package cmd

import (
	"github.com/spf13/cobra"

	"github.com/clouddrove/smurf/internal/ai"
)

// --ai is offered by 42 commands, and each was expected to call
// ai.AIExplainError itself on every error path. An AST audit of
// internal/{docker,helm,terraform} found 19 functions where that had not
// happened: three never called it at all and sixteen missed some of their
// returns. `smurf stf validate --ai` printed nothing on a validation failure,
// which is the failure the flag exists for.
//
// Per-site calls will drift again, because nothing makes the next early return
// remember. Wrapping RunE once covers every command and every path, including
// paths added later, and leaves one place to reason about.
//
// The existing inline calls are left where they are. ai.AIExplainError reports
// only the first failure in a process, so an inline call still wins for the
// commands that have one and this wrapper covers the rest, rather than a
// large mechanical edit through code with little test coverage.

// wireAIExplain walks the command tree and makes every command that offers
// --ai explain its own failures.
//
// It runs from Execute rather than an init function because subcommands
// register through their own init functions, and their order relative to this
// package's is not defined. By the time Execute is called the tree is whole.
func wireAIExplain(root *cobra.Command) {
	for _, c := range root.Commands() {
		wireAIExplain(c)

		// A command with Run rather than RunE cannot report a failure, and the
		// group commands are the only ones in that position.
		if c.RunE == nil {
			continue
		}

		// The flag is deliberately not inspected here. Cobra merges a parent's
		// persistent flags into a child's flag set during ParseFlags, which
		// happens inside Execute, after this runs. Checking now would see only
		// local flags, so declaring --ai once as a persistent flag on the three
		// groups, an obvious future tidy-up, would silently stop every command
		// being wrapped, with nothing failing to say so. Deferring the lookup
		// to run time means the flags are parsed by the time it is read.
		inner := c.RunE
		c.RunE = func(cc *cobra.Command, args []string) error {
			err := inner(cc, args)
			if err == nil {
				return nil
			}
			if useAIFlag(cc) {
				ai.AIExplainError(true, err.Error())
			}
			return err
		}
	}
}

// useAIFlag reports whether --ai is set on the command being executed,
// checking inherited flags as well as local ones so a persistent declaration
// works.
func useAIFlag(c *cobra.Command) bool {
	if c == nil {
		return false
	}
	if f := c.Flags().Lookup("ai"); f != nil {
		if v, err := c.Flags().GetBool("ai"); err == nil {
			return v
		}
	}
	if f := c.InheritedFlags().Lookup("ai"); f != nil {
		if v, err := c.InheritedFlags().GetBool("ai"); err == nil {
			return v
		}
	}
	return false
}
