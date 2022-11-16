package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/md5"
	"encoding/hex"

	"github.com/databricks/bricks/git"
	"github.com/databricks/bricks/project"
)

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
	// hostname of the workspace this snapshot is for
	Host string `json:"host"`
	// Path in workspace for project repo
	RemotePath string `json:"remote_path"`
	// Map of all files present in the remote repo with the:
	// key: relative file path from project root
	// value: last time the remote instance of this file was updated
	LastUpdatedTimes map[string]time.Time `json:"last_modified_times"`
	// This map maps local file names of notebooks to their remote names
	// eg. notebook named "foo.py" locally would be stored as "foo", thus this
	// map will contain an entry "foo.py" -> "foo"
	// 
	// Contains all databricks notebooks that have been synced atleast once
	NotebookLocalToRemoteNames map[string]string `json:"notebook_local_to_remote_names"`
}

type diff struct {
	put    []string
	delete []string
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
func (s *Snapshot) getPath(ctx context.Context) (string, error) {
	prj := project.Get(ctx)
	cacheDir, err := prj.CacheDir()
	if err != nil {
		return "", err
	}
	snapshotDir := filepath.Join(cacheDir, syncSnapshotDirName)
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		err = os.Mkdir(snapshotDir, os.ModeDir|os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("failed to create config directory: %s", err)
		}
	}
	fileName := GetFileName(s.Host, s.RemotePath)
	return filepath.Join(snapshotDir, fileName), nil
}

func newSnapshot(ctx context.Context, remotePath string) (*Snapshot, error) {
	prj := project.Get(ctx)

	// Get host this snapshot is for
	wsc := prj.WorkspacesClient()

	// TODO: The host may be late-initialized in certain Azure setups where we
	// specify the workspace by its resource ID. tracked in: https://databricks.atlassian.net/browse/DECO-194
	host := wsc.Config.Host
	if host == "" {
		return nil, fmt.Errorf("failed to resolve host for snapshot")
	}

	return &Snapshot{
		Host:                       host,
		RemotePath:                 remotePath,
		LastUpdatedTimes:           make(map[string]time.Time),
		NotebookLocalToRemoteNames: make(map[string]string),
	}, nil
}

func (s *Snapshot) storeSnapshot(ctx context.Context) error {
	snapshotPath, err := s.getPath(ctx)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(snapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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

func (s *Snapshot) loadSnapshot(ctx context.Context) error {
	snapshotPath, err := s.getPath(ctx)
	if err != nil {
		return err
	}
	// Snapshot file not found. We do not load anything
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return nil
	}

	f, err := os.Open(snapshotPath)
	if err != nil {
		return fmt.Errorf("failed to open persisted sync snapshot file: %s", err)
	}
	defer f.Close()

	bytes, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read sync snapshot from disk: %s", err)
	}
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return fmt.Errorf("failed to json unmarshal persisted snapshot: %s", err)
	}
	return nil
}

func (d diff) IsEmpty() bool {
	return len(d.put) == 0 && len(d.delete) == 0
}

func (d diff) String() string {
	if d.IsEmpty() {
		return "no changes"
	}
	var changes []string
	if len(d.put) > 0 {
		changes = append(changes, fmt.Sprintf("PUT: %s", strings.Join(d.put, ", ")))
	}
	if len(d.delete) > 0 {
		changes = append(changes, fmt.Sprintf("DELETE: %s", strings.Join(d.delete, ", ")))
	}
	return strings.Join(changes, ", ")
}

func (s Snapshot) diff(all []git.File) (change diff) {
	currentFilenames := map[string]bool{}
	lastModifiedTimes := s.LastUpdatedTimes
	for _, f := range all {
		// create set of current files to figure out if removals are needed
		currentFilenames[f.Relative] = true
		// get current modified timestamp
		modified := f.Modified()
		lastSeenModified, seen := lastModifiedTimes[f.Relative]

		if !seen || modified.After(lastSeenModified) {
			change.put = append(change.put, f.Relative)
			lastModifiedTimes[f.Relative] = modified
		}
	}
	// figure out files in the snapshot, but not on local filesystem
	for relative := range lastModifiedTimes {
		_, exists := currentFilenames[relative]
		if exists {
			continue
		}
		// add them to a delete batch
		change.delete = append(change.delete, relative)
		// remove the file from snapshot
		delete(lastModifiedTimes, relative)
	}
	// and remove them from the snapshot
	for _, v := range change.delete {
		delete(lastModifiedTimes, v)
	}
	return
}
