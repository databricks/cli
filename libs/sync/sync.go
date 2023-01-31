package sync

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/databricks/bricks/libs/git"
	"github.com/databricks/bricks/libs/sync/repofiles"
	"github.com/databricks/databricks-sdk-go"
)

type SyncOptions struct {
	LocalPath  string
	RemotePath string

	Full bool

	SnapshotBasePath string

	PollInterval time.Duration

	WorkspaceClient *databricks.WorkspaceClient

	Host string
}

type Sync struct {
	*SyncOptions

	fileSet   *git.FileSet
	snapshot  *Snapshot
	repoFiles *repofiles.RepoFiles
}

// New initializes and returns a new [Sync] instance.
func New(ctx context.Context, opts SyncOptions) (*Sync, error) {
	fileSet, err := git.NewFileSet(opts.LocalPath)
	if err != nil {
		return nil, err
	}
	err = fileSet.EnsureValidGitIgnoreExists()
	if err != nil {
		return nil, err
	}

	// Verify that the remote path we're about to synchronize to is valid and allowed.
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

	// For full sync, we start with an empty snapshot.
	// For incremental sync, we try to load an existing snapshot to start from.
	var snapshot *Snapshot
	if opts.Full {
		snapshot, err = newSnapshot(&opts)
		if err != nil {
			return nil, fmt.Errorf("unable to instantiate new sync snapshot: %w", err)
		}
	} else {
		snapshot, err = loadOrNewSnapshot(&opts)
		if err != nil {
			return nil, fmt.Errorf("unable to load sync snapshot: %w", err)
		}
	}

	repoFiles := repofiles.Create(opts.RemotePath, opts.LocalPath, opts.WorkspaceClient)

	return &Sync{
		SyncOptions: &opts,

		fileSet:   fileSet,
		snapshot:  snapshot,
		repoFiles: repoFiles,
	}, nil
}

func (s *Sync) RunOnce(ctx context.Context) error {
	repoFiles := repofiles.Create(s.RemotePath, s.LocalPath, s.WorkspaceClient)
	applyDiff := syncCallback(ctx, repoFiles)

	// tradeoff: doing portable monitoring only due to macOS max descriptor manual ulimit setting requirement
	// https://github.com/gorakhargosh/watchdog/blob/master/src/watchdog/observers/kqueue.py#L394-L418
	all, err := s.fileSet.All()
	if err != nil {
		log.Printf("[ERROR] cannot list files: %s", err)
		return err
	}

	change, err := s.snapshot.diff(all)
	if err != nil {
		return err
	}
	if change.IsEmpty() {
		return nil
	}

	log.Printf("[INFO] Action: %v", change)
	err = applyDiff(change)
	if err != nil {
		return err
	}

	err = s.snapshot.Save(ctx)
	if err != nil {
		log.Printf("[ERROR] cannot store snapshot: %s", err)
		return err
	}

	return nil
}

func (s *Sync) RunContinuous(ctx context.Context) error {
	var once sync.Once

	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := s.RunOnce(ctx)
			if err != nil {
				return err
			}

			once.Do(func() {
				log.Printf("[INFO] Initial Sync Complete")
			})
		}
	}
}
