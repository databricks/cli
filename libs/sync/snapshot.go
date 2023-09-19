package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"crypto/md5"
	"encoding/hex"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/log"
)

// Bump it up every time a potentially breaking change is made to the snapshot schema
const LatestSnapshotVersion = "v1"

// A snapshot is a persistant store of knowledge this CLI has about state of files
// in the remote repo. We use the last modified times (mtime) of files to determine
// whether a files need to be updated in the remote repo.
//
// 1. Any stale files in the remote repo are updated. That is if the last modified
// time recorded in the snapshot is less than the actual last modified time of the file
//
// 2. Any files present in snapshot but absent locally are deleted from remote path
//
// Changing either the databricks workspace (ie Host) or the remote path (ie RemotePath)
// local files are being synced to will make this CLI switch to a different
// snapshot for persisting/loading sync state
type Snapshot struct {
	// Path where this snapshot was loaded from and will be saved to.
	// Intentionally not part of the snapshot state because it may be moved by the user.
	SnapshotPath string `json:"-"`

	// New indicates if this is a fresh snapshot or if it was loaded from disk.
	New bool `json:"-"`

	// version for snapshot schema. Only snapshots matching the latest snapshot
	// schema version are used and older ones are invalidated (by deleting them)
	Version string `json:"version"`

	// hostname of the workspace this snapshot is for
	Host string `json:"host"`

	// Path in workspace for project repo
	RemotePath string `json:"remote_path"`

	*FilesState
}

const syncSnapshotDirName = "sync-snapshots"

func GetFileName(host, remotePath string) string {
	hash := md5.Sum([]byte(host + remotePath))
	hashString := hex.EncodeToString(hash[:])
	return hashString[:16] + ".json"
}

// Compute path of the snapshot file on the local machine
// The file name for unique for a tuple of (host, remotePath)
// precisely it's the first 16 characters of md5(concat(host, remotePath))
func SnapshotPath(opts *SyncOptions) (string, error) {
	snapshotDir := filepath.Join(opts.SnapshotBasePath, syncSnapshotDirName)
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		err = os.MkdirAll(snapshotDir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create config directory: %s", err)
		}
	}
	fileName := GetFileName(opts.Host, opts.RemotePath)
	return filepath.Join(snapshotDir, fileName), nil
}

func newSnapshot(ctx context.Context, opts *SyncOptions) (*Snapshot, error) {
	path, err := SnapshotPath(opts)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		SnapshotPath: path,
		New:          true,

		Version:    LatestSnapshotVersion,
		Host:       opts.Host,
		RemotePath: opts.RemotePath,
		FilesState: &FilesState{
			LastModifiedTimes:  make(map[string]time.Time),
			LocalToRemoteNames: make(map[string]string),
			RemoteToLocalNames: make(map[string]string),
		},
	}, nil
}

func (s *Snapshot) Save(ctx context.Context) error {
	f, err := os.OpenFile(s.SnapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create/open persisted sync snapshot file: %s", err)
	}
	defer f.Close()

	// persist snapshot to disk
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to json marshal in-memory snapshot: %s", err)
	}
	_, err = f.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to write sync snapshot to disk: %s", err)
	}
	return nil
}

func (s *Snapshot) Destroy(ctx context.Context) error {
	err := os.Remove(s.SnapshotPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to destroy sync snapshot file: %s", err)
	}
	return nil
}

func loadOrNewSnapshot(ctx context.Context, opts *SyncOptions) (*Snapshot, error) {
	snapshot, err := newSnapshot(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Snapshot file not found. We return the new copy.
	if _, err := os.Stat(snapshot.SnapshotPath); os.IsNotExist(err) {
		return snapshot, nil
	}

	bytes, err := os.ReadFile(snapshot.SnapshotPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sync snapshot from disk: %s", err)
	}

	var fromDisk Snapshot
	err = json.Unmarshal(bytes, &fromDisk)
	if err != nil {
		return nil, fmt.Errorf("failed to json unmarshal persisted snapshot: %s", err)
	}

	// invalidate old snapshot with schema versions
	if fromDisk.Version != LatestSnapshotVersion {
		log.Warnf(ctx, "Did not load existing snapshot because its version is %s while the latest version is %s", snapshot.Version, LatestSnapshotVersion)
		return newSnapshot(ctx, opts)
	}

	// unmarshal again over the existing snapshot instance
	err = json.Unmarshal(bytes, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to json unmarshal persisted snapshot: %s", err)
	}

	snapshot.New = false
	return snapshot, nil
}

// TODO: resolve disparity with pointers vs structs here.
// TODO: Consolidate structs in the sync library a bit more.

func (s *Snapshot) diff(ctx context.Context, all []fileset.File) (change diff, err error) {
	targetState, err := toFilesState(ctx, all)
	if err != nil {

		return diff{}, fmt.Errorf("error while computing new sync state: %w", err)
	}

	currentState := s.FilesState
	if err := currentState.validate(); err != nil {
		return diff{}, fmt.Errorf("error parsing existing sync state: %w", err)
	}

	// Compute change operations to get from current state to new target state.
	d := computeDiff(targetState, currentState)

	// Update state to new value. This is not persisted to the file system before
	// the changes are actually applied successfully.
	s.FilesState = targetState
	return *d, nil
}
