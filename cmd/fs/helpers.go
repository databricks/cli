package fs

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/filer/completer"
	"github.com/spf13/cobra"
)

func filerForPath(ctx context.Context, fullPath string) (filer.Filer, string, error) {
	// Split path at : to detect any file schemes
	parts := strings.SplitN(fullPath, ":", 2)

	// If no scheme is specified, then local path
	if len(parts) < 2 {
		f, err := filer.NewLocalClient("")
		return f, fullPath, err
	}

	// On windows systems, paths start with a drive letter. If the scheme
	// is a single letter and the OS is windows, then we conclude the path
	// is meant to be a local path.
	if runtime.GOOS == "windows" && len(parts[0]) == 1 {
		f, err := filer.NewLocalClient("")
		return f, fullPath, err
	}

	if parts[0] != "dbfs" {
		return nil, "", fmt.Errorf("invalid scheme: %s", parts[0])
	}

	path := parts[1]
	w := cmdctx.WorkspaceClient(ctx)

	// If the specified path has the "Volumes" prefix, use the Files API.
	if strings.HasPrefix(path, "/Volumes/") {
		f, err := filer.NewFilesClient(w, "/")
		return f, path, err
	}

	// The file is a dbfs file, and uses the DBFS APIs
	f, err := filer.NewDbfsClient(w, "/")
	return f, path, err
}

const dbfsPrefix string = "dbfs:"

func isDbfsPath(path string) bool {
	return strings.HasPrefix(path, dbfsPrefix)
}

type validArgs struct {
	mustWorkspaceClientFunc func(cmd *cobra.Command, args []string) error
	filerForPathFunc        func(ctx context.Context, fullPath string) (filer.Filer, string, error)
	pathArgCount            int
	onlyDirs                bool
}

func newValidArgs() *validArgs {
	return &validArgs{
		mustWorkspaceClientFunc: root.MustWorkspaceClient,
		filerForPathFunc:        filerForPath,
		pathArgCount:            1,
		onlyDirs:                false,
	}
}

func (v *validArgs) Validate(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cmd.SetContext(root.SkipPrompt(cmd.Context()))

	if len(args) >= v.pathArgCount {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	err := v.mustWorkspaceClientFunc(cmd, args)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	filer, toCompletePath, err := v.filerForPathFunc(cmd.Context(), toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	completer := completer.New(cmd.Context(), filer, v.onlyDirs)

	// Dbfs should have a prefix and always use the "/" separator
	isDbfsPath := isDbfsPath(toComplete)
	if isDbfsPath {
		completer.SetPrefix(dbfsPrefix)
		completer.SetIsLocalPath(false)
	}

	completions, directive, err := completer.CompletePath(toCompletePath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return completions, directive
}
