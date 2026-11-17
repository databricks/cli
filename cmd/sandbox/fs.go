package sandbox

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	filesystem "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

// sandboxCpLong replaces the help of cp, which describes DBFS and UC paths,
// and keeps its description of how directories are copied.
const sandboxCpLong = `Copy files and directories between your local filesystem and sandboxes.

  Use <sandbox-id>:/absolute/path for sandbox paths.

  Recursively copying a directory will copy all files inside directory
  at SOURCE_PATH to the directory at TARGET_PATH.

  When copying a file, if TARGET_PATH is a directory, the file will be created
  inside the directory, otherwise the file is created at TARGET_PATH.

  Files that already exist at the target are skipped unless --overwrite is set.
`

// sandboxFsExamples are shown in each command's help.
var sandboxFsExamples = map[string]string{
	"cp": `  databricks sandbox fs cp ./file happy-panda-1234:/tmp/file
  databricks sandbox fs cp --overwrite ./file happy-panda-1234:/tmp/file
  databricks sandbox fs cp -r happy-panda-1234:/tmp/dir ./local/dir`,
	"ls":    `  databricks sandbox fs ls happy-panda-1234:/tmp`,
	"cat":   `  databricks sandbox fs cat happy-panda-1234:/tmp/file`,
	"mkdir": `  databricks sandbox fs mkdir happy-panda-1234:/tmp/a/b`,
	"rm":    `  databricks sandbox fs rm -r happy-panda-1234:/tmp/dir`,
}

func newFsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "Filesystem related commands",
	}
	filesystem.AddCommands(cmd, sandboxFilerForPath)
	cmd.Long = `Transfer files between your local filesystem and sandboxes.

Use <sandbox-id>:/absolute/path for sandbox paths. A sandbox name from the
local cache populated by sandbox list also works in place of the ID. Paths
without a sandbox prefix refer to the local filesystem.`
	for _, sub := range cmd.Commands() {
		sub.Long = sub.Short + " Use <sandbox-id>:/absolute/path for sandbox paths."
		if sub.Name() == "cp" {
			sub.Long = sandboxCpLong
		}
		sub.Example = sandboxFsExamples[sub.Name()]
		sub.ValidArgsFunction = completeSandboxPaths(sub.ValidArgsFunction)
	}
	cmd.Example = `  databricks sandbox fs cp -r happy-panda-1234:/foo/bar ./local/output
  databricks sandbox fs cp ./file happy-panda-1234:/tmp/file --overwrite
  databricks sandbox fs ls happy-panda-1234:/tmp`
	return cmd
}

func sandboxFilerForPath(ctx context.Context, fullPath string) (filer.Filer, string, error) {
	identifier, name := filesystem.SplitPath(fullPath)
	if identifier == "" && name == fullPath {
		f, err := filer.NewLocalClient("")
		return f, name, err
	}
	if identifier == "" {
		return nil, "", errors.New("empty sandbox ID: use <sandbox-id>:/path")
	}
	if !path.IsAbs(name) {
		return nil, "", fmt.Errorf("sandbox path must be absolute: use %s:/path", identifier)
	}
	w := cmdctx.WorkspaceClient(ctx)
	id, err := resolveLocalID(ctx, cacheProfile(w), identifier)
	if err != nil {
		return nil, "", err
	}
	f, err := filer.NewSandboxClient(w, id, "/")
	return f, cleanKeepingTrailingSlash(name), err
}

// cleanKeepingTrailingSlash cleans p but keeps a trailing slash, which shell
// completion needs to list a directory's entries instead of its siblings.
func cleanKeepingTrailingSlash(p string) string {
	cleaned := path.Clean(p)
	if strings.HasSuffix(p, "/") && cleaned != "/" {
		cleaned += "/"
	}
	return cleaned
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
		for _, s := range getSandboxes(cmd.Context(), cacheProfile(w)) {
			completions = append(completions, s.ID+":/")
			if s.Name != "" && s.Name != s.ID {
				completions = append(completions, s.Name+":/")
			}
		}
		return completions, directive
	}
}
