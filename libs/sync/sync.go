package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/libs/sync/repofiles"
	"github.com/databricks/databricks-sdk-go"
)

type SyncOptions struct {
	LocalPath  string
	RemotePath string

	PersistSnapshot bool

	SnapshotBasePath string

	PollInterval time.Duration

	WorkspaceClient *databricks.WorkspaceClient

	Host string
}

type Sync struct {
	*SyncOptions

	fileSet *git.FileSet
}

// New initializes and returns a new [Sync] instance.
func New(ctx context.Context, opts SyncOptions) (*Sync, error) {
	fileSet := git.NewFileSet(opts.LocalPath)
	err := fileSet.EnsureValidGitIgnoreExists()
	if err != nil {
		return nil, err
	}

	// Retrieve current user so that we can verify that the remote path
	// is nested under the user's directories.
	err = ensureRemotePathIsUsable(ctx, opts.WorkspaceClient, opts.RemotePath)
	if err != nil {
		return nil, err
	}

	// TODO: The host may be late-initialized in certain Azure setups where we
	// specify the workspace by its resource ID. tracked in: https://databricks.atlassian.net/browse/DECO-194
	opts.Host = opts.WorkspaceClient.Config.Host
	if opts.Host == "" {
		return nil, fmt.Errorf("failed to resolve host for snapshot")
	}

	return &Sync{
		SyncOptions: &opts,
		fileSet:     fileSet,
	}, nil
}

// RunWatchdog kicks off a polling loop to monitor local changes and synchronize
// them to the remote workspace path.
func (s *Sync) RunWatchdog(ctx context.Context) error {
	repoFiles := repofiles.Create(s.RemotePath, s.LocalPath, s.WorkspaceClient)
	syncCallback := syncCallback(ctx, repoFiles)
	return spawnWatchdog(ctx, syncCallback, s)
}
