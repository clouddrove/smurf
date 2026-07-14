package stf

import (
	"context"
	"time"

	"github.com/clouddrove/smurf/internal/terraform"
	"github.com/spf13/cobra"
)

// completionTimeout bounds the `terraform show` call used by dynamic shell
// completion below. The underlying terraform process is killed as soon as
// the context expires, so a slow or unreachable remote backend can't hang
// the user's shell while they're typing a command.
const completionTimeout = 2 * time.Second

// completeStateAddresses is a cobra ValidArgsFunction that suggests resource
// addresses currently tracked in the Terraform state, for `state-rm`. It
// never prints or prompts, and degrades to no completions on any error
// (uninitialized working directory, missing terraform binary, unreachable
// backend, timeout, etc) rather than ever erroring the shell.
//
// state-rm accepts one or more addresses (cobra.MinimumNArgs(1)), so
// completion stays available for every argument position, not just the
// first.
func completeStateAddresses(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	dir, _ := cmd.Flags().GetString("dir")

	ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
	defer cancel()

	addresses, err := terraform.StateResourceAddresses(ctx, dir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return addresses, cobra.ShellCompDirectiveNoFileComp
}

// Note on `import`'s ADDRESS argument: unlike state-rm, the address that
// import takes identifies a resource block in configuration that is NOT YET
// in state (import adds it). Completing it from the existing state list
// would suggest exactly the wrong set of addresses, and there is no
// reusable helper for enumerating not-yet-imported config addresses (that
// would require parsing the HCL configuration, which doesn't exist in
// internal/terraform today). So import's ADDRESS/ID arguments intentionally
// get no dynamic completion here; falls back to normal file completion.

func init() {
	stateRmCmd.ValidArgsFunction = completeStateAddresses
}
