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
		if c.Flags().Lookup("ai") == nil {
			continue
		}

		inner := c.RunE
		cmdRef := c
		c.RunE = func(cc *cobra.Command, args []string) error {
			err := inner(cc, args)
			if err == nil {
				return nil
			}
			// Read the flag from the command that owns it. Reading it from cc
			// would miss the value when cobra passes a different receiver.
			useAI, flagErr := cmdRef.Flags().GetBool("ai")
			if flagErr == nil && useAI {
				ai.AIExplainError(true, err.Error())
			}
			return err
		}
	}
}
