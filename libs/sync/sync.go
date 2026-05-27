package sync

import (
	"context"
	"errors"
	"fmt"
	stdsync "sync"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/set"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type OutputHandler func(context.Context, <-chan Event)

type SyncOptions struct {
	WorktreeRoot vfs.Path
	LocalRoot    vfs.Path
	Paths        []string
	Include      []string
	Exclude      []string

	RemotePath string

	Full bool

	SnapshotBasePath string

	PollInterval time.Duration

	WorkspaceClient *databricks.WorkspaceClient

	CurrentUser *iam.User

	Host string

	OutputHandler OutputHandler

	DryRun bool
}

// Sync runs file synchronization in three layers:
//
//  1. Discovery (libs/git, libs/fileset): walks the local tree and produces a
//     list of files to consider, honoring include/exclude rules.
//
//  2. Snapshot diff (libs/sync/snapshot.go, libs/sync/diff.go): compares the
//     discovered files against a local snapshot of mtimes from the previous
//     run and produces a diff (puts, deletes, mkdirs, rmdirs) — the action
//     plan for this iteration. If no snapshot exists, every file becomes a
//     put.
//
//  3. Remote filter (libs/sync/remote_filter.go): an optional pre-flight that
//     fetches content SHAs from the workspace and drops puts whose remote SHA
//     already matches the local SHA. We only run it when the snapshot is
//     fresh (no prior state), which is the only case where Layer 2 produces
//     false-positive puts at scale (e.g. on a CI runner). When a local
//     snapshot exists, Layer 2 is already accurate enough; paying for a
//     bulk remote list would be wasted work.
type Sync struct {
	*SyncOptions

	fileSet        *git.FileSet
	includeFileSet *fileset.FileSet
	excludeFileSet *fileset.FileSet

	snapshot     *Snapshot
	filer        filer.Filer
	remoteFilter *RemoteFilter

	// Synchronization progress events are sent to this event notifier.
	notifier EventNotifier
	seq      int

	// WaitGroup is automatically created when an output handler is provided in the SyncOptions.
	// Close call is required to ensure the output handler goroutine handles all events in time.
	outputWaitGroup *stdsync.WaitGroup
}

// New initializes and returns a new [Sync] instance.
func New(ctx context.Context, opts SyncOptions) (*Sync, error) {
	fileSet, err := git.NewFileSet(ctx, opts.WorktreeRoot, opts.LocalRoot, opts.Paths)
	if err != nil {
		return nil, err
	}

	WriteGitIgnore(ctx, opts.LocalRoot.Native())

	includeFileSet, err := fileset.NewGlobSet(opts.LocalRoot, opts.Include)
	if err != nil {
		return nil, err
	}

	excludeFileSet, err := fileset.NewGlobSet(opts.LocalRoot, opts.Exclude)
	if err != nil {
		return nil, err
	}

	// Verify that the remote path we're about to synchronize to is valid and allowed.
	err = EnsureRemotePathIsUsable(ctx, opts.WorkspaceClient, opts.RemotePath, opts.CurrentUser)
	if err != nil {
		return nil, err
	}

	// TODO: The host may be late-initialized in certain Azure setups where we
	// specify the workspace by its resource ID. tracked in: https://databricks.atlassian.net/browse/DECO-194
	opts.Host = opts.WorkspaceClient.Config.Host
	if opts.Host == "" {
		return nil, errors.New("failed to resolve host for snapshot")
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

	filerImpl, err := filer.NewWorkspaceFilesClient(opts.WorkspaceClient, opts.RemotePath)
	if err != nil {
		return nil, err
	}

	// The remote SHA list call is not part of the Filer interface (it's a
	// sync-only optimization, not a general filesystem op), so we type-assert
	// the concrete client. In tests we plug in a stub via NewWithRemoteFilter.
	var remoteFilter *RemoteFilter
	if wfc, ok := filerImpl.(*filer.WorkspaceFilesClient); ok {
		remoteFilter = NewRemoteFilter(wfc, opts.RemotePath)
	}

	var notifier EventNotifier
	outputWaitGroup := &stdsync.WaitGroup{}
	if opts.OutputHandler != nil {
		ch := make(chan Event, MaxRequestsInFlight)
		notifier = &ChannelNotifier{ch}
		outputWaitGroup.Go(func() {
			opts.OutputHandler(ctx, ch)
		})
	} else {
		notifier = &NopNotifier{}
	}

	return &Sync{
		SyncOptions: &opts,

		fileSet:         fileSet,
		includeFileSet:  includeFileSet,
		excludeFileSet:  excludeFileSet,
		snapshot:        snapshot,
		filer:           filerImpl,
		remoteFilter:    remoteFilter,
		notifier:        notifier,
		outputWaitGroup: outputWaitGroup,
		seq:             0,
	}, nil
}

func (s *Sync) Close() {
	if s.notifier == nil {
		return
	}
	s.notifier.Close()
	s.notifier = nil
	s.outputWaitGroup.Wait()
}

func (s *Sync) notifyStart(ctx context.Context, d diff) {
	// If this is not the initial iteration we can ignore no-ops.
	if s.seq > 0 && d.IsEmpty() {
		return
	}
	s.notifier.Notify(ctx, newEventStart(s.seq, d.put, d.delete, s.DryRun))
}

func (s *Sync) notifyProgress(ctx context.Context, action EventAction, path string, progress float32) {
	s.notifier.Notify(ctx, newEventProgress(s.seq, action, path, progress, s.DryRun))
}

func (s *Sync) notifyComplete(ctx context.Context, d diff) {
	// If this is not the initial iteration we can ignore no-ops.
	if s.seq > 0 && d.IsEmpty() {
		return
	}
	s.notifier.Notify(ctx, newEventComplete(s.seq, d.put, d.delete, s.DryRun))
	s.seq++
}

// Upload all files in the file tree rooted at the local path configured in the
// SyncOptions to the remote path configured in the SyncOptions.
//
// Returns the list of files tracked (and synchronized) by the syncer during the run,
// and an error if any occurred.
func (s *Sync) RunOnce(ctx context.Context) ([]fileset.File, error) {
	// Layer 1: discovery.
	files, err := s.GetFileList(ctx)
	if err != nil {
		return files, err
	}

	// Layer 2: snapshot-driven action plan.
	change, err := s.snapshot.diff(ctx, files)
	if err != nil {
		return files, err
	}

	// Layer 3: remote-state filter, only when the snapshot is fresh.
	// With an existing snapshot, Layer 2 is precise; with no snapshot, every
	// file is a put — so we ask the workspace what's already there and drop
	// puts whose contents already match.
	if s.snapshot.New && s.remoteFilter != nil {
		change = s.remoteFilter.Apply(ctx, change, files, s.snapshot.LocalToRemoteNames)
	}

	s.notifyStart(ctx, change)
	if change.IsEmpty() {
		s.notifyComplete(ctx, change)
		return files, nil
	}

	err = s.applyDiff(ctx, change)
	if err != nil {
		return files, err
	}

	if !s.DryRun {
		err = s.snapshot.Save(ctx)
		if err != nil {
			log.Errorf(ctx, "cannot store snapshot: %s", err)
			return files, err
		}
	}

	s.notifyComplete(ctx, change)
	return files, nil
}

func (s *Sync) GetFileList(ctx context.Context) ([]fileset.File, error) {
	// tradeoff: doing portable monitoring only due to macOS max descriptor manual ulimit setting requirement
	// https://github.com/gorakhargosh/watchdog/blob/master/src/watchdog/observers/kqueue.py#L394-L418
	all := set.NewSetF(func(f fileset.File) string {
		return f.Relative
	})
	gitFiles, err := s.fileSet.Files()
	if err != nil {
		log.Errorf(ctx, "cannot list files: %s", err)
		return nil, err
	}
	all.Add(gitFiles...)

	include, err := s.includeFileSet.Files()
	if err != nil {
		log.Errorf(ctx, "cannot list include files: %s", err)
		return nil, err
	}

	all.Add(include...)

	exclude, err := s.excludeFileSet.Files()
	if err != nil {
		log.Errorf(ctx, "cannot list exclude files: %s", err)
		return nil, err
	}

	for _, f := range exclude {
		all.Remove(f)
	}

	return all.Iter(), nil
}

func (s *Sync) RunContinuous(ctx context.Context) error {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, err := s.RunOnce(ctx)
			if err != nil {
				return err
			}
		}
	}
}
