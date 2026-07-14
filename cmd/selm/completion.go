package selm

import (
	"context"
	"time"

	"github.com/clouddrove/smurf/internal/helm"
	"github.com/spf13/cobra"
)

// completionTimeout bounds the internal Helm/Kubernetes calls used by
// dynamic shell completion below, so a slow or unreachable cluster can't
// hang the user's shell while they're typing a command.
const completionTimeout = 2 * time.Second

// completeReleaseNames is a cobra ValidArgsFunction that suggests existing
// Helm release names for the first positional argument of commands that
// operate on an already-deployed release (upgrade, uninstall, status,
// history, rollback). It never prints or prompts, and degrades to no
// completions on any error so a broken kubeconfig or unreachable cluster
// never breaks the user's shell.
//
// It is wired up in each command's own init() (next to where its flags are
// registered), not here, because RegisterFlagCompletionFunc/ValidArgsFunction
// must run after the target command's own flags exist; centralizing the
// wiring in this file would run before those flags are registered, since Go
// runs init() functions in file name order and "completion.go" sorts before
// most of the command files.
func completeReleaseNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		// Release name already given (or being followed by a chart path /
		// revision number); nothing useful to complete from release names.
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	namespace, _ := cmd.Flags().GetString("namespace")

	names, err := helm.ListReleaseNames(namespace, completionTimeout)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeNamespaces is a cobra flag completion function that suggests
// existing Kubernetes namespace names for --namespace/-n. It never prints or
// prompts, and degrades to no completions on any error (missing kubeconfig,
// unreachable cluster, timeout, etc).
func completeNamespaces(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
	defer cancel()

	names, err := helm.ListNamespaces(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}
