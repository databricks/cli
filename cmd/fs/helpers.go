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

		completions, dirPath, directive := completer.CompletePath(path)

		for i := range completions {
			completions[i] = prependDirPath(dirPath, completions[i], !isDbfsPath)

			if isDbfsPath {
				// The dirPath will start with a "/", so we'll prefix the completions with
				// the selectedPrefix without a trailing "/"
				prefix := dbfsPrefix[:len(dbfsPrefix)-1]
				completions[i] = fmt.Sprintf("%s%s", prefix, completions[i])
			} else {
				completions[i] = handleLocalPrefix(toComplete, completions[i])
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

// Prepend directory path to completion name:
// - Don't add a separator if the directory path is empty
// - Don't add a separator if the dir path already ends with a separator
// - Support \ as separator for local Windows paths
// Note that we don't use path utilities to concatenate the path and name
// because we want to preserve the formatting of whatever path the user has
// typed (e.g. //a/b///c)
func prependDirPath(dirPath string, completion string, isLocalPath bool) string {
	defaultSeparator := "/"
	windowsSeparator := "\\"

	isLocalWindowsPath := isLocalPath && runtime.GOOS == "windows"

	separator := ""
	if isLocalWindowsPath && len(dirPath) > 0 &&
		!strings.HasSuffix(dirPath, defaultSeparator) &&
		!strings.HasSuffix(dirPath, windowsSeparator) {
		separator = windowsSeparator
	} else if !isLocalWindowsPath && len(dirPath) > 0 &&
		!strings.HasSuffix(dirPath, defaultSeparator) {
		separator = defaultSeparator
	}

	return fmt.Sprintf("%s%s%s", dirPath, separator, completion)
}

// Drop the local prefix from completions if the path to complete doesn't
// start with it. We do this because the local filer returns paths in the
// current folder with the local prefix (./ (and .\ on windows))
func handleLocalPrefix(toComplete string, completion string) string {
	shouldDrop := shouldDropLocalPrefix(toComplete, completion, "./") ||
		shouldDropLocalPrefix(toComplete, completion, ".\\")

	if shouldDrop {
		return completion[2:]
	}

	return completion
}

func shouldDropLocalPrefix(toComplete string, completion string, localPrefix string) bool {
	return !strings.HasPrefix(toComplete, localPrefix) && strings.HasPrefix(completion, localPrefix)
}
