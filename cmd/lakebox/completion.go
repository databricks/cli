package lakebox

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

// completeSandboxIDs is a Cobra ValidArgsFunction returning the caller's
// sandbox IDs (and display names, when distinct from the ID) for tab
// completion. Reads purely from the local cache populated by `lakebox
// list` / `create` / `status` / etc. — no API call on `<TAB>`, so a flaky
// network or unrefreshed auth token never makes the shell hang.
//
// Cobra runs ValidArgsFunction in a separate process from the main
// command, so we still need to bootstrap the workspace client just to
// learn which profile we're under. The cache itself is profile-scoped.
//
// Best-effort: any failure (no profile resolvable, empty cache) returns
// no suggestions instead of an error so the shell stays usable and the
// user can still type the ID by hand.
func completeSandboxIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	profile := completionProfile(cmd, args)
	if profile == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	sbs := getSandboxes(cmd.Context(), profile)
	if len(sbs) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	// Offer the ID always, plus the display name when the user actually
	// set one (server defaults `name` to the sandbox ID, so don't echo
	// the same string twice).
	suggestions := make([]string, 0, len(sbs)*2)
	for _, s := range sbs {
		suggestions = append(suggestions, s.ID)
		if s.Name != "" && s.Name != s.ID {
			suggestions = append(suggestions, s.Name)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeSSHKeyHashes is the equivalent for `ssh-key delete <hash>`,
// returning the hashes of registered keys. SSH-key hashes aren't cached
// locally (per-user cap is ~100 server-side and listing is cheap), so
// this path still calls the API.
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

// completionProfile returns the profile name the cache is keyed under,
// matching the resolution used by the runtime commands (Profile if set,
// else Host). Returns "" if the workspace client can't be bootstrapped.
func completionProfile(cmd *cobra.Command, args []string) string {
	if err := root.MustWorkspaceClient(cmd, args); err != nil {
		return ""
	}
	w := cmdctx.WorkspaceClient(cmd.Context())
	if w.Config.Profile != "" {
		return w.Config.Profile
	}
	return w.Config.Host
}
