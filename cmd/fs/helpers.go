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
const volumesPefix string = "dbfs:/Volumes"
const localPefix string = "./"

func isDbfsPath(path string) bool {
	return strings.HasPrefix(path, dbfsPrefix)
}

func getValidArgsFunction(
	pathArgCount int,
	onlyDirs bool,
	filerForPathFunc func(ctx context.Context, fullPath string) (filer.Filer, string, error),
	mustWorkspaceClientFunc func(cmd *cobra.Command, args []string) error,
) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cmd.SetContext(root.SkipPrompt(cmd.Context()))

		err := mustWorkspaceClientFunc(cmd, args)

		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		filer, path, err := filerForPathFunc(cmd.Context(), toComplete)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		isDbfsPath := isDbfsPath(toComplete)

		wsc := root.WorkspaceClient(cmd.Context())
		_, err = wsc.CurrentUser.Me(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		completer := completer.NewCompleter(cmd.Context(), filer, onlyDirs)

		if len(args) >= pathArgCount {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		completions, directive := completer.CompleteRemotePath(path)

		for i, completion := range completions {
			if isDbfsPath {
				// The completions will start with a "/", so we'll prefix them with the
				// selectedPrefix without a trailing "/"
				prefix := dbfsPrefix[:len(dbfsPrefix)-1]
				completions[i] = fmt.Sprintf("%s%s", prefix, completion)
			} else if shouldDropLocalPrefix(toComplete, completion) {

				completions[i] = completion[len(localPefix):]
			} else {
				completions[i] = completion
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
}

// Drop the local prefix from completions if the path to complete doesn't
// start with it. We do this because the local filer returns paths in the
// current folder with the local prefix (./).
func shouldDropLocalPrefix(toComplete string, completion string) bool {
	return !strings.HasPrefix(toComplete, localPefix) && strings.HasPrefix(completion, localPefix)
}
