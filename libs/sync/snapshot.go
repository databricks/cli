package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/md5"
	"encoding/hex"

	"github.com/databricks/bricks/libs/fileset"
	"github.com/databricks/bricks/libs/notebook"
)

// Bump it up every time a potentially breaking change is made to the snapshot schema
const LatestSnapshotVersion = "v1"

// A snapshot is a persistant store of knowledge bricks cli has about state of files
// in the remote repo. We use the last modified times (mtime) of files to determine
// whether a files need to be updated in the remote repo.
//
// 1. Any stale files in the remote repo are updated. That is if the last modified
// time recorded in the snapshot is less than the actual last modified time of the file
//
// 2. Any files present in snapshot but absent locally are deleted from remote path
//
// Changing either the databricks workspace (ie Host) or the remote path (ie RemotePath)
// local files are being synced to will make bricks cli switch to a different
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

	// Map of all files present in the remote repo with the:
	// key: relative file path from project root
	// value: last time the remote instance of this file was updated
	LastUpdatedTimes map[string]time.Time `json:"last_modified_times"`

	// This map maps local file names to their remote names
	// eg. notebook named "foo.py" locally would be stored as "foo", thus this
	// map will contain an entry "foo.py" -> "foo"
	LocalToRemoteNames map[string]string `json:"local_to_remote_names"`

	// Inverse of localToRemoteNames. Together the form a bijective mapping (ie
	// there is a 1:1 unique mapping between local and remote name)
	RemoteToLocalNames map[string]string `json:"remote_to_local_names"`
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

func newSnapshot(opts *SyncOptions) (*Snapshot, error) {
	path, err := SnapshotPath(opts)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		SnapshotPath: path,
		New:          true,

		Version:            LatestSnapshotVersion,
		Host:               opts.Host,
		RemotePath:         opts.RemotePath,
		LastUpdatedTimes:   make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
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

func loadOrNewSnapshot(opts *SyncOptions) (*Snapshot, error) {
	snapshot, err := newSnapshot(opts)
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
		log.Printf("Did not load existing snapshot because its version is %s while the latest version is %s", snapshot.Version, LatestSnapshotVersion)
		return newSnapshot(opts)
	}

	// unmarshal again over the existing snapshot instance
	err = json.Unmarshal(bytes, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to json unmarshal persisted snapshot: %s", err)
	}

	snapshot.New = false
	return snapshot, nil
}

func (s *Snapshot) diff(all []fileset.File) (change diff, err error) {
	lastModifiedTimes := s.LastUpdatedTimes
	remoteToLocalNames := s.RemoteToLocalNames
	localToRemoteNames := s.LocalToRemoteNames

	// set of files currently present in the local file system and tracked by git
	localFileSet := map[string]struct{}{}
	for _, f := range all {
		localFileSet[f.Relative] = struct{}{}
	}

	for _, f := range all {
		// get current modified timestamp
		modified := f.Modified()
		lastSeenModified, seen := lastModifiedTimes[f.Relative]

		if !seen || modified.After(lastSeenModified) {
			lastModifiedTimes[f.Relative] = modified

			// change separators to '/' for file paths in remote store
			unixFileName := filepath.ToSlash(f.Relative)

			// put file in databricks workspace
			change.put = append(change.put, unixFileName)

			// get file metadata about whether it's a notebook
			isNotebook, _, err := notebook.Detect(f.Absolute)
			if err != nil {
				return change, err
			}

			// Strip extension for notebooks.
			remoteName := unixFileName
			if isNotebook {
				ext := filepath.Ext(remoteName)
				remoteName = strings.TrimSuffix(remoteName, ext)
			}

			// If the remote handle of a file changes, we want to delete the old
			// remote version of that file to avoid duplicates.
			// This can happen if a python notebook is converted to a python
			// script or vice versa
			oldRemoteName, ok := localToRemoteNames[f.Relative]
			if ok && oldRemoteName != remoteName {
				change.delete = append(change.delete, oldRemoteName)
				delete(remoteToLocalNames, oldRemoteName)
			}

			// We cannot allow two local files in the project to point to the same
			// remote path
			prevLocalName, ok := remoteToLocalNames[remoteName]
			_, prevLocalFileExists := localFileSet[prevLocalName]
			if ok && prevLocalName != f.Relative && prevLocalFileExists {
				return change, fmt.Errorf("both %s and %s point to the same remote file location %s. Please remove one of them from your local project", prevLocalName, f.Relative, remoteName)
			}
			localToRemoteNames[f.Relative] = remoteName
			remoteToLocalNames[remoteName] = f.Relative
		}
	}
	// figure out files in the snapshot.lastModifiedTimes, but not on local
	// filesystem. These will be deleted
	for localName := range lastModifiedTimes {
		_, exists := localFileSet[localName]
		if exists {
			continue
		}

		// TODO: https://databricks.atlassian.net/browse/DECO-429
		// Add error wrapper giving instructions like this for all errors here :)
		remoteName, ok := localToRemoteNames[localName]
		if !ok {
			return change, fmt.Errorf("missing remote path for local path: %s. Please try syncing again after deleting .databricks/sync-snapshots dir from your project root", localName)
		}

		// add them to a delete batch
		change.delete = append(change.delete, remoteName)
	}
	// and remove them from the snapshot
	for _, remoteName := range change.delete {
		// we do note assert that remoteName exists in remoteToLocalNames since it
		// will be missing for files with remote name changed
		localName := remoteToLocalNames[remoteName]

		delete(lastModifiedTimes, localName)
		delete(remoteToLocalNames, remoteName)
		delete(localToRemoteNames, localName)
	}
	return
}
