package sync

import (
	"context"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

// List directories or repos under the specified path.
// Returns a channel such that we can list multiple paths in parallel.
func fetchDirs(ctx context.Context, wsc *databricks.WorkspaceClient, path string) <-chan []string {
	ch := make(chan []string, 1)
	go func() {
		defer close(ch)

		files, err := wsc.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
			Path: path,
		})
		if err != nil {
			return
		}

		// Filter directories and repos.
		// We're interested only in paths we can sync to.
		var dirs []string
		for _, file := range files {
			switch file.ObjectType {
			case workspace.ObjectTypeDirectory, workspace.ObjectTypeRepo:
				dirs = append(dirs, file.Path)
			}
		}

		ch <- dirs
	}()

	return ch
}

func completeRemotePath(
	ctx context.Context,
	wsc *databricks.WorkspaceClient,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	me, err := wsc.CurrentUser.Me(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	prefixes := []string{
		path.Clean("/Users/"+me.UserName) + "/",
		path.Clean("/Repos/"+me.UserName) + "/",
	}

	validPrefix := false
	for _, p := range prefixes {
		if strings.HasPrefix(toComplete, p) {
			validPrefix = true
		}
	}

	if !validPrefix {
		return prefixes, cobra.ShellCompDirectiveNoSpace
	}

	// If the user is TAB-ing their way through a path, the path in `toComplete`
	// is valid and we should list nested directories.
	// If the path in `toComplete` is incomplete, however,
	// then we should list adjacent directories.
	nested := fetchDirs(ctx, wsc, toComplete)
	adjacent := fetchDirs(ctx, wsc, path.Dir(toComplete))
	dirs := <-nested
	if dirs == nil {
		dirs = <-adjacent
	}

	return dirs, cobra.ShellCompDirectiveNoSpace
}
