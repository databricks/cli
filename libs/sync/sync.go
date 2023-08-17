package sync

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type SyncOptions struct {
	LocalPath  string
	RemotePath string
	Include    []string
	Exclude    []string

	Full bool

	SnapshotBasePath string

	PollInterval time.Duration

	WorkspaceClient *databricks.WorkspaceClient

	CurrentUser *iam.User

	Host string
}

type Sync struct {
	*SyncOptions

	fileSet        *git.FileSet
	includeFileSet *fileset.GlobSet
	excludeFileSet *fileset.GlobSet

	snapshot *Snapshot
	filer    filer.Filer

	// Synchronization progress events are sent to this event notifier.
	notifier EventNotifier
	seq      int
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

	includeFileSet := fileset.NewGlobSet(opts.LocalPath, opts.Include)
	excludeFileSet := fileset.NewGlobSet(opts.LocalPath, opts.Exclude)

	// Verify that the remote path we're about to synchronize to is valid and allowed.
	err = EnsureRemotePathIsUsable(ctx, opts.WorkspaceClient, opts.RemotePath, opts.CurrentUser)
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
		snapshot, err = newSnapshot(ctx, &opts)
		if err != nil {
			return nil, fmt.Errorf("unable to instantiate new sync snapshot: %w", err)
		}
	} else {
		snapshot, err = loadOrNewSnapshot(ctx, &opts)
		if err != nil {
			return nil, fmt.Errorf("unable to load sync snapshot: %w", err)
		}
	}

	filer, err := filer.NewWorkspaceFilesClient(opts.WorkspaceClient, opts.RemotePath)
	if err != nil {
		return nil, err
	}

	return &Sync{
		SyncOptions: &opts,

		fileSet:        fileSet,
		includeFileSet: includeFileSet,
		excludeFileSet: excludeFileSet,
		snapshot:       snapshot,
		filer:          filer,
		notifier:       &NopNotifier{},
		seq:            0,
	}, nil
}

func (s *Sync) Events() <-chan Event {
	ch := make(chan Event, MaxRequestsInFlight)
	s.notifier = &ChannelNotifier{ch}
	return ch
}

func (s *Sync) Close() {
	if s.notifier == nil {
		return
	}
	s.notifier.Close()
	s.notifier = nil
}

func (s *Sync) notifyStart(ctx context.Context, d diff) {
	// If this is not the initial iteration we can ignore no-ops.
	if s.seq > 0 && d.IsEmpty() {
		return
	}
	s.notifier.Notify(ctx, newEventStart(s.seq, d.put, d.delete))
}

func (s *Sync) notifyProgress(ctx context.Context, action EventAction, path string, progress float32) {
	s.notifier.Notify(ctx, newEventProgress(s.seq, action, path, progress))
}

func (s *Sync) notifyComplete(ctx context.Context, d diff) {
	// If this is not the initial iteration we can ignore no-ops.
	if s.seq > 0 && d.IsEmpty() {
		return
	}
	s.notifier.Notify(ctx, newEventComplete(s.seq, d.put, d.delete))
	s.seq++
}

func (s *Sync) RunOnce(ctx context.Context) error {
	// tradeoff: doing portable monitoring only due to macOS max descriptor manual ulimit setting requirement
	// https://github.com/gorakhargosh/watchdog/blob/master/src/watchdog/observers/kqueue.py#L394-L418
	all := make([]fileset.File, 0)
	gitFiles, err := s.fileSet.All()
	if err != nil {
		log.Errorf(ctx, "cannot list files: %s", err)
		return err
	}
	all = append(all, gitFiles...)

	include, err := s.includeFileSet.All()
	if err != nil {
		log.Errorf(ctx, "cannot list include files: %s", err)
		return err
	}

	// Avoiding duplicates with Git tracked and include files
	for _, i := range include {
		if slices.ContainsFunc(all, func(a fileset.File) bool {
			return a.Absolute == i.Absolute
		}) {
			continue
		}

		all = append(all, i)
	}

	all = append(all, include...)

	exclude, err := s.excludeFileSet.All()
	if err != nil {
		log.Errorf(ctx, "cannot list exclude files: %s", err)
		return err
	}

	files := make([]fileset.File, 0)
	for _, f := range all {
		if slices.ContainsFunc(exclude, func(a fileset.File) bool {
			return a.Absolute == f.Absolute
		}) {
			continue
		}

		files = append(files, f)
	}

	change, err := s.snapshot.diff(ctx, files)
	if err != nil {
		return err
	}

	s.notifyStart(ctx, change)
	if change.IsEmpty() {
		s.notifyComplete(ctx, change)
		return nil
	}

	err = s.applyDiff(ctx, change)
	if err != nil {
		return err
	}

	err = s.snapshot.Save(ctx)
	if err != nil {
		log.Errorf(ctx, "cannot store snapshot: %s", err)
		return err
	}

	s.notifyComplete(ctx, change)
	return nil
}

func (s *Sync) DestroySnapshot(ctx context.Context) error {
	return s.snapshot.Destroy(ctx)
}

func (s *Sync) SnapshotPath() string {
	return s.snapshot.SnapshotPath
}

func (s *Sync) RunContinuous(ctx context.Context) error {
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
		}
	}
}
