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
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/spf13/cobra"
)

// SplitPath separates a remote prefix from a path, preserving local Windows drive paths.
func SplitPath(fullPath string) (scheme, name string) {
	// Split path at : to detect any file schemes
	parts := strings.SplitN(fullPath, ":", 2)

	// If no scheme is specified, then local path
	if len(parts) < 2 {
		return "", fullPath
	}

	// On windows systems, paths start with a drive letter. If the scheme
	// is a single letter and the OS is windows, then we conclude the path
	// is meant to be a local path.
	if runtime.GOOS == "windows" && len(parts[0]) == 1 {
		return "", fullPath
	}
	return parts[0], parts[1]
}

// isUCPath reports whether a path without its scheme is a UC Volumes or UC Skills path.
func isUCPath(path string) bool {
	return strings.HasPrefix(path, "/Volumes/") || strings.HasPrefix(path, "/Skills/")
}

// warnSchemelessUCPaths warns once when any of the given paths starts with
// /Volumes/ or /Skills/ without a scheme. Such a path is usually a UC path
// missing its scheme, but on macOS /Volumes/ is also the mount point for local
// drives, so it stays a local path. The warning makes sure a forgotten scheme
// never acts on the local filesystem silently.
func warnSchemelessUCPaths(ctx context.Context, fullPaths ...string) {
	var local []string
	for _, fullPath := range fullPaths {
		if scheme, _ := SplitPath(fullPath); scheme == "" && isUCPath(fullPath) {
			local = append(local, fullPath)
		}
	}
	if len(local) == 0 {
		return
	}
	log.Warnf(ctx, "Paths without a scheme are local paths: %s. For a Unity Catalog volume or skill, add the dbfs: scheme", strings.Join(local, ", "))
}

func filerForPath(ctx context.Context, fullPath string) (filer.Filer, string, error) {
	scheme, path := SplitPath(fullPath)
	if scheme == "" && path == fullPath {
		f, err := filer.NewLocalClient("")
		return f, path, err
	}

	if scheme != "dbfs" {
		return nil, "", fmt.Errorf("invalid scheme: %s", scheme)
	}

	w := cmdctx.WorkspaceClient(ctx)

	// If the specified path has the "Volumes" or "Skills" prefix, use the Files API.
	// Otherwise the file is a dbfs file, and uses the DBFS APIs.
	var f filer.Filer
	var err error
	if isUCPath(path) {
		f, err = filer.NewFilesClient(ctx, w, "/")
	} else {
		f, err = filer.NewDbfsClient(w, "/")
	}
	if err != nil {
		return nil, "", err
	}

	// Writes are not retried here: a write without overwrite may fail on the file its own
	// earlier attempt created, and replaying a streamed source needs buffering. The Files API
	// filer (Volumes) retries its own uploads when the source is seekable or small enough
	// to buffer; DBFS writes get one attempt.
	return retry.FilerExceptWrites(f), path, nil
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
	scheme, _ := SplitPath(toComplete)
	if scheme != "" {
		completer.SetPrefix(scheme + ":")
		completer.SetIsLocalPath(false)
	}

	completions, directive, err := completer.CompletePath(toCompletePath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return completions, directive
}
