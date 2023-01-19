package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/cmd/sync/repofiles"
	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func matchesBasePaths(me *scim.User, path string) error {
	basePaths := []string{
		fmt.Sprintf("/Users/%s/", me.UserName),
		fmt.Sprintf("/Repos/%s/", me.UserName),
	}
	basePathMatch := false
	for _, basePath := range basePaths {
		if strings.HasPrefix(path, basePath) {
			basePathMatch = true
			break
		}
	}
	if !basePathMatch {
		return fmt.Errorf("path must be nested under %s or %s", basePaths[0], basePaths[1])
	}
	return nil
}

// ensureRemotePathIsUsable checks if the specified path is nested under
// expected base paths and if it is a directory or repository.
func ensureRemotePathIsUsable(ctx context.Context, wsc *databricks.WorkspaceClient, me *scim.User, path string) error {
	err := matchesBasePaths(me, path)
	if err != nil {
		return err
	}

	// Ensure that the remote path exits.
	// If it is a repo, it has to exist.
	// If it is a workspace path, it may not exist.
	info, err := wsc.Workspace.GetStatusByPath(ctx, path)
	if err != nil {
		// We only deal with 404s below.
		if !apierr.IsMissing(err) {
			return err
		}

		switch {
		case strings.HasPrefix(path, "/Repos/"):
			return fmt.Errorf("%s does not exist; please create it first", path)
		case strings.HasPrefix(path, "/Users/"):
			// The workspace path doesn't exist. Create it and try again.
			err = wsc.Workspace.MkdirsByPath(ctx, path)
			if err != nil {
				return fmt.Errorf("unable to create directory at %s: %w", path, err)
			}
			return ensureRemotePathIsUsable(ctx, wsc, me, path)
		default:
			return err
		}
	}

	log.Printf(
		"[DEBUG] Path %s has type %s (ID: %d)",
		info.Path,
		strings.ToLower(info.ObjectType.String()),
		info.ObjectId,
	)

	// We expect the object at path to be a directory or a repo.
	switch info.ObjectType {
	case workspace.ObjectTypeDirectory:
		return nil
	case workspace.ObjectTypeRepo:
		return nil
	}

	return fmt.Errorf("%s points to a %s", path, strings.ToLower(info.ObjectType.String()))
}

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "run syncs for the project",

	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prj := project.Get(ctx)
		wsc := prj.WorkspacesClient()

		me, err := prj.Me()
		if err != nil {
			return err
		}

		if *remotePath == "" {
			repositoryName, err := git.RepositoryName()
			if err != nil {
				return err
			}
			*remotePath = fmt.Sprintf("/Repos/%s/%s", me.UserName, repositoryName)
		}

		log.Printf("[INFO] Remote file sync location: %v", *remotePath)
		err = ensureRemotePathIsUsable(ctx, wsc, me, *remotePath)
		if err != nil {
			return err
		}

		root := prj.Root()
		repoFiles := repofiles.Create(*remotePath, root, wsc)
		syncCallback := syncCallback(ctx, repoFiles)
		err = spawnWatchdog(ctx, *interval, syncCallback, *remotePath)
		return err
	},
}

// project files polling interval
var interval *time.Duration

var remotePath *string

var persistSnapshot *bool

func init() {
	root.RootCmd.AddCommand(syncCmd)
	interval = syncCmd.Flags().Duration("interval", 1*time.Second, "project files polling interval")
	remotePath = syncCmd.Flags().String("remote-path", "", "remote path to store repo in. eg: /Repos/me@example.com/test-repo")
	persistSnapshot = syncCmd.Flags().Bool("persist-snapshot", true, "whether to store local snapshots of sync state")
}
