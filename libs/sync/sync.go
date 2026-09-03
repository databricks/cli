package sync

import (
	"context"
	"errors"
	"fmt"
	stdsync "sync"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

type OutputHandler func(context.Context, <-chan Event)

// FileCounts reports how many files a sync run uploaded and deleted. A file
// whose remote name changed (e.g. a script converted to a notebook) is counted
// as both an upload and a delete, because that is what the sync performs.
type FileCounts struct {
	Uploaded int
	Deleted  int
}

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

	NoValidateRemotePath bool
}

type Sync struct {
	*SyncOptions

	fileList *FileList

	snapshot *Snapshot
	filer    filer.Filer

	// Synchronization progress events are sent to this event notifier.
	notifier EventNotifier
	seq      int

	// fileCounts holds the counts from the most recent run. Kept here rather than
	// derived from the event stream because the notifier is a no-op unless the
	// caller supplies an OutputHandler.
	fileCounts FileCounts

	// WaitGroup is automatically created when an output handler is provided in the SyncOptions.
	// Close call is required to ensure the output handler goroutine handles all events in time.
	outputWaitGroup *stdsync.WaitGroup
}

// New initializes and returns a new [Sync] instance.
func New(ctx context.Context, opts SyncOptions) (*Sync, error) {
	fileList, err := NewFileList(ctx, opts.WorktreeRoot, opts.LocalRoot, opts.Paths, opts.Include, opts.Exclude)
	if err != nil {
		return nil, err
	}

	WriteGitIgnore(ctx, opts.LocalRoot.Native())

	if !opts.NoValidateRemotePath {
		// Verify that the remote path we're about to synchronize to is valid and allowed.
		err = EnsureRemotePathIsUsable(ctx, opts.WorkspaceClient, opts.RemotePath, opts.CurrentUser, opts.DryRun)
		if err != nil {
			return nil, err
		}
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

	filer, err := filer.NewWorkspaceFilesClient(opts.WorkspaceClient, opts.RemotePath)
	if err != nil {
		return nil, err
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

		fileList:        fileList,
		snapshot:        snapshot,
		filer:           filer,
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
	files, err := s.GetFileList(ctx)
	if err != nil {
		return files, err
	}

	change, err := s.snapshot.diff(ctx, files)
	if err != nil {
		return files, err
	}
	s.fileCounts = FileCounts{Uploaded: len(change.put), Deleted: len(change.delete)}

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

// FileCounts returns the upload and delete counts from the most recent run.
func (s *Sync) FileCounts() FileCounts {
	return s.fileCounts
}

// GetFileList returns the local files selected for sync, without syncing them.
func (s *Sync) GetFileList(ctx context.Context) ([]fileset.File, error) {
	return s.fileList.Files(ctx)
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
