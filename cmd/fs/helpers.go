package fs

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/completer"
	"github.com/databricks/cli/libs/filer"
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
	w := root.WorkspaceClient(ctx)

	// If the specified path has the "Volumes" prefix, use the Files API.
	if strings.HasPrefix(path, "/Volumes/") {
		f, err := filer.NewFilesClient(w, "/")
		return f, path, err
	}

	// The file is a dbfs file, and uses the DBFS APIs
	f, err := filer.NewDbfsClient(w, "/")
	return f, path, err
}

const dbfsPrefix string = "dbfs:/"
const volumesPefix string = "dbfs:/Volumes"

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

	err := v.mustWorkspaceClientFunc(cmd, args)

	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	filer, toCompletePath, err := v.filerForPathFunc(cmd.Context(), toComplete)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	wsc := root.WorkspaceClient(cmd.Context())
	_, err = wsc.CurrentUser.Me(cmd.Context())
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	completer := completer.New(cmd.Context(), filer, v.onlyDirs)

	if len(args) >= v.pathArgCount {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	completions, dirPath, directive := completer.CompletePath(toCompletePath)

	isDbfsPath := isDbfsPath(toComplete)

	for i := range completions {
		completions[i] = filepath.Join(dirPath, completions[i])

		if isDbfsPath {
			// Add dbfs:/ prefix to completions
			completions[i] = path.Join(dbfsPrefix, completions[i])
		} else {
			// Clean up ./ (and .\ on Windows) from local completions
			completions[i] = filepath.Clean(completions[i])
		}
	}

	// If the path is a dbfs path, we try to add the dbfs:/Volumes prefix suggestion
	if isDbfsPath && strings.HasPrefix(volumesPefix, toComplete) {
		completions = append(completions, volumesPefix)
	}

	// If the path is local, we try to add the dbfs:/ prefix suggestion
	if !isDbfsPath && strings.HasPrefix(dbfsPrefix, toComplete) {
		completions = append(completions, dbfsPrefix)
	}

	return completions, directive
}
