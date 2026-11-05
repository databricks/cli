package sandbox

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"

	filesystem "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

func newFsCommand() *cobra.Command {
	cmd := filesystem.NewWithPathResolver(sandboxFilerForPath)
	cmd.GroupID = ""
	cmd.Long = `Transfer files between your local filesystem and sandboxes.

Use SANDBOX:/absolute/path for sandbox paths. SANDBOX can be an ID or a
name from the local cache populated by sandbox list. Paths without a
sandbox prefix refer to the local filesystem.`
	for _, sub := range cmd.Commands() {
		sub.Long = sub.Short + " Use SANDBOX:/absolute/path for sandbox paths."
		sub.ValidArgsFunction = completeSandboxPaths(sub.ValidArgsFunction)
	}
	cmd.Example = `  databricks sandbox fs cp -r my-sandbox:/foo/bar ./local/output
  databricks sandbox fs cp ./file my-sandbox:/tmp/file --overwrite
  databricks sandbox fs ls my-sandbox:/tmp`
	return cmd
}

func sandboxFilerForPath(ctx context.Context, fullPath string) (filer.Filer, string, error) {
	identifier, name := filesystem.SplitPath(fullPath)
	if identifier == "" && name == fullPath {
		f, err := filer.NewLocalClient("")
		return f, name, err
	}
	if identifier == "" {
		return nil, "", errors.New("empty sandbox identifier: use SANDBOX:/path")
	}
	if !path.IsAbs(name) {
		return nil, "", fmt.Errorf("sandbox path must be absolute: use %s:/path", identifier)
	}
	w := cmdctx.WorkspaceClient(ctx)
	profile := w.Config.Profile
	if profile == "" {
		profile = w.Config.Host
	}
	id, err := resolveLocalID(ctx, profile, identifier)
	if err != nil {
		return nil, "", err
	}
	f, err := filer.NewSandboxClient(w, id, "/")
	return f, path.Clean(name), err
}

func completeSandboxPaths(complete cobra.CompletionFunc) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		completions, directive := complete(cmd, args, toComplete)
		identifier, _ := filesystem.SplitPath(toComplete)
		if identifier != "" || directive != cobra.ShellCompDirectiveNoSpace {
			return completions, directive
		}
		completions = slices.DeleteFunc(completions, func(p string) bool { return p == "dbfs:/" })
		w := cmdctx.WorkspaceClient(cmd.Context())
		profile := w.Config.Profile
		if profile == "" {
			profile = w.Config.Host
		}
		for _, s := range getSandboxes(cmd.Context(), profile) {
			completions = append(completions, s.ID+":/")
			if s.Name != "" && s.Name != s.ID {
				completions = append(completions, s.Name+":/")
			}
		}
		return completions, directive
	}
}
