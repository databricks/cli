package fs

import (
	"context"
	"fmt"
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

func isDbfsPath(path string) bool {
	return strings.HasPrefix(path, dbfsPrefix)
}

func getValidArgsFunction(pathArgCount int, onlyDirs bool, filerForPathFunc func(ctx context.Context, fullPath string) (filer.Filer, string, error)) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		fmt.Println("GetValidArgsFunction")
		cmd.SetContext(root.SkipPrompt(cmd.Context()))

		wsc := root.WorkspaceClient(cmd.Context())
		if wsc == nil {
			return nil, cobra.ShellCompDirectiveError
		}

		isValidPrefix := isDbfsPath(toComplete)

		if !isValidPrefix {
			return []string{dbfsPrefix}, cobra.ShellCompDirectiveNoSpace
		}

		filer, path, err := filerForPathFunc(cmd.Context(), toComplete)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		_, err = wsc.CurrentUser.Me(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		completer := completer.NewCompleter(cmd.Context(), filer, onlyDirs)

		if len(args) < pathArgCount {
			completions, directive := completer.CompleteRemotePath(path)

			// The completions will start with a "/", so we'll prefix
			// them with the dbfsPrefix without a trailing "/" (dbfs:)
			prefix := dbfsPrefix[:len(dbfsPrefix)-1]

			// Add the prefix to the completions
			for i, completion := range completions {
				completions[i] = fmt.Sprintf("%s%s", prefix, completion)
			}

			return completions, directive
		} else {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}
}
