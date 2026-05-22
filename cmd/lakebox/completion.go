package lakebox

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

// completeSandboxIDs is a Cobra ValidArgsFunction returning the caller's
// sandbox IDs for tab completion. Cobra runs this in a separate process
// from the main command, so we need to bootstrap the workspace client
// ourselves (PreRunE is skipped during completion).
//
// Best-effort: any failure (no auth, no network, lakebox not deployed)
// returns no suggestions instead of an error so the shell stays usable
// and the user can still type the ID by hand.
func completeSandboxIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if err := root.MustWorkspaceClient(cmd, args); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx := cmd.Context()
	api, err := newLakeboxAPI(cmdctx.WorkspaceClient(ctx))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	entries, err := api.list(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.SandboxID)
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

// completeSSHKeyHashes is the equivalent for `ssh-key delete <hash>`,
// returning the hashes of registered keys. Same best-effort semantics.
func completeSSHKeyHashes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if err := root.MustWorkspaceClient(cmd, args); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx := cmd.Context()
	api, err := newLakeboxAPI(cmdctx.WorkspaceClient(ctx))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	keys, err := api.listKeys(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	hashes := make([]string, 0, len(keys))
	for _, k := range keys {
		hashes = append(hashes, k.KeyHash)
	}
	return hashes, cobra.ShellCompDirectiveNoFileComp
}
