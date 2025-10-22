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

type Sync struct {
	*SyncOptions

	fileSet        *git.FileSet
	includeFileSet *fileset.FileSet
	excludeFileSet *fileset.FileSet

	snapshot *Snapshot
	filer    filer.Filer

	// Synchronization progress events are sent to this event notifier.
	notifier EventNotifier
	seq      int

	// WaitGroup is automatically created when an output handler is provided in the SyncOptions.
	// Close call is required to ensure the output handler goroutine handles all events in time.
	outputWaitGroup *stdsync.WaitGroup
}

// New initializes and returns a new [Sync] instance.
func New(ctx context.Context, opts SyncOptions) (*Sync, error) {
	fileSet, err := git.NewFileSet(opts.WorktreeRoot, opts.LocalRoot, opts.Paths)
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

	filer, err := filer.NewWorkspaceFilesClient(opts.WorkspaceClient, opts.RemotePath)
	if err != nil {
		return nil, err
	}

	var notifier EventNotifier
	outputWaitGroup := &stdsync.WaitGroup{}
	if opts.OutputHandler != nil {
		ch := make(chan Event, MaxRequestsInFlight)
		notifier = &ChannelNotifier{ch}
		outputWaitGroup.Add(1)
		go func() {
			defer outputWaitGroup.Done()
			opts.OutputHandler(ctx, ch)
		}()
	} else {
		notifier = &NopNotifier{}
	}

	return &Sync{
		SyncOptions: &opts,

		fileSet:         fileSet,
		includeFileSet:  includeFileSet,
		excludeFileSet:  excludeFileSet,
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

// FileList contains the list of files to sync and counts of files by their sync status.
type FileList struct {
	Files                 []fileset.File
	Included              int
	ExcludedDirectories   int
	ExcludedFiles         int
	ExcludedBySyncExclude int
}

// Upload all files in the file tree rooted at the local path configured in the
// SyncOptions to the remote path configured in the SyncOptions.
//
// Returns the file list with counts and an error if any occurred.
func (s *Sync) RunOnce(ctx context.Context) (*FileList, error) {
	fileList, err := s.GetFileList(ctx)
	if err != nil {
		return fileList, err
	}

	change, err := s.snapshot.diff(ctx, fileList.Files)
	if err != nil {
		return fileList, err
	}

	s.notifyStart(ctx, change)
	if change.IsEmpty() {
		s.notifyComplete(ctx, change)
		return fileList, nil
	}

	err = s.applyDiff(ctx, change)
	if err != nil {
		return fileList, err
	}

	if !s.DryRun {
		err = s.snapshot.Save(ctx)
		if err != nil {
			log.Errorf(ctx, "cannot store snapshot: %s", err)
			return fileList, err
		}
	}

	s.notifyComplete(ctx, change)
	return fileList, nil
}

func (s *Sync) GetFileList(ctx context.Context) (*FileList, error) {
	// Get ignorer for exclude patterns
	var excludeIgnorer fileset.Ignorer
	if s.excludeFileSet != nil {
		excludeIgnorer = s.excludeFileSet.Ignorer()
	}

	// Check if we have include patterns that might override gitignore
	hasIncludePatterns := false
	if s.includeFileSet != nil {
		ignorer := s.includeFileSet.Ignorer()
		hasIncludePatterns = ignorer != nil
	}

	// Build set for easier manipulation
	syncSet := set.NewSetF(func(f fileset.File) string {
		return f.Relative
	})

	var excludedDirs int
	var excludedFiles int

	// Get files respecting gitignore
	files, stats, err := s.fileSet.FilesWithStats()
	if err != nil {
		log.Errorf(ctx, "cannot list files: %s", err)
		return nil, err
	}
	syncSet.Add(files...)
	excludedDirs = stats.SkippedDirectories
	excludedFiles = stats.SkippedFiles

	// If there are include patterns, also add files matching those patterns
	// (even if they're gitignored). This allows include patterns to override gitignore.
	if hasIncludePatterns {
		includeFiles, err := s.includeFileSet.Files()
		if err != nil {
			log.Errorf(ctx, "cannot list files matching include patterns: %s", err)
			return nil, err
		}
		syncSet.Add(includeFiles...)
	}

	// Remove files matching exclude patterns and count them
	// We can post-filter exclude patterns since they don't affect directory traversal
	excludedBySyncExclude := 0
	if excludeIgnorer != nil {
		for _, f := range syncSet.Iter() {
			ignored, err := excludeIgnorer.IgnoreFile(f.Relative)
			if err != nil {
				log.Errorf(ctx, "cannot check if file matches exclude pattern: %s", err)
				return nil, err
			}
			// If not ignored by excluder, it means the file matches exclude patterns
			if !ignored {
				syncSet.Remove(f)
				excludedBySyncExclude++
			}
		}
	}

	return &FileList{
		Files:                 syncSet.Iter(),
		Included:              syncSet.Size(),
		ExcludedDirectories:   excludedDirs,
		ExcludedFiles:         excludedFiles,
		ExcludedBySyncExclude: excludedBySyncExclude,
	}, nil
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
