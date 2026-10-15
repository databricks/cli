package sync

import (
	"context"
	"path"
	"strings"

	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/libs/workspaceapi"
	"github.com/spf13/cobra"
)

// List directories or repos under the specified path.
// Returns a channel such that we can list multiple paths in parallel.
func fetchDirs(ctx context.Context, wsc *sdkclients.Clients, path string) <-chan []string {
	ch := make(chan []string, 1)
	go func() {
		defer close(ch)

		files, err := wsc.Workspace.ListAll(ctx, workspaceapi.ListWorkspaceRequest{
			Path: path,
		})
		if err != nil {
			return
		}

		// Filter directories and repos.
		// We're interested only in paths we can sync to.
		var dirs []string
		for _, file := range files {
			if file.ObjectType == workspaceapi.ObjectTypeDirectory || file.ObjectType == workspaceapi.ObjectTypeRepo {
				dirs = append(dirs, file.Path)
			}
		}

		ch <- dirs
	}()

	return ch
}

func completeRemotePath(
	ctx context.Context,
	wsc *sdkclients.Clients,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	me, err := iamutil.GetCurrentUser(ctx, wsc)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	prefixes := []string{
		path.Clean("/Users/"+utils.Deref(me.UserName)) + "/",
		path.Clean("/Repos/"+utils.Deref(me.UserName)) + "/",
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
