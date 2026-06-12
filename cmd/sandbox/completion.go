package sandbox

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

// completeSandboxIDs returns sandbox IDs and (distinct) display names
// from the local cache for tab completion. Cache-only so an unrefreshed
// token never hangs the shell; any failure yields no suggestions.
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
	// Server defaults `name` to the ID, so only emit the name when it's distinct.
	suggestions := make([]string, 0, len(sbs)*2)
	for _, s := range sbs {
		suggestions = append(suggestions, s.ID)
		if s.Name != "" && s.Name != s.ID {
			suggestions = append(suggestions, s.Name)
		}
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completeSSHKeyHashes returns registered key hashes for `ssh-key delete`.
// Hashes aren't cached locally, so this path calls the API.
func completeSSHKeyHashes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if err := root.MustWorkspaceClient(cmd, args); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx := cmd.Context()
	api, err := newSandboxAPI(cmdctx.WorkspaceClient(ctx))
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
