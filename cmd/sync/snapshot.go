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

// type snapshot map[string]time.Time

// TODO: add comments here
type snapshot struct {
	Host              string               `json:"host"`
	UserName          string               `json:"user_name"`
	LastModifiedTimes map[string]time.Time `json:"last_modified_times"`
}

type diff struct {
	put    []string
	delete []string
}

const syncSnapshotDirName = "sync-snapshots"

// Compute path of the snapshot file on the local machine
// The file name for unique for a tuple of (host, username)
// precisely it's the first 8 characters of md5(concat(host, username))
func (s *snapshot) getPath(ctx context.Context) (string, error) {
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
	hash := md5.Sum([]byte(s.Host + s.UserName))
	hashString := hex.EncodeToString(hash[:])
	return filepath.Join(snapshotDir, "sync-"+hashString[:8]+".json"), nil
}

func newSnapshot(ctx context.Context) (snapshot, error) {
	prj := project.Get(ctx)

	// Get host this snapshot is for
	wsc := prj.WorkspacesClient()
	if wsc == nil {
		return snapshot{}, fmt.Errorf("failed to resolve workspaces client for project")
	}
	host := wsc.Config.Host
	if host == "" {
		return snapshot{}, fmt.Errorf("failed to resolve host for snapshot")
	}

	// Get username this snapshot
	me, err := prj.Me()
	if err != nil {
		return snapshot{}, err
	}
	userName := me.UserName
	if userName == "" {
		return snapshot{}, fmt.Errorf("failed to resolve user name for snapshot")
	}

	return snapshot{
		Host:              host,
		UserName:          userName,
		LastModifiedTimes: make(map[string]time.Time),
	}, nil
}

func (s *snapshot) storeSnapshot(ctx context.Context) error {
	snapshotPath, err := s.getPath(ctx)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(snapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create/open persisted sync snapshot file: %s", err)
	}
	defer f.Close()

	// persist snapshot to disk
	bytes, err := json.MarshalIndent(s.LastModifiedTimes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to json marshal in-memory snapshot: %s", err)
	}
	_, err = f.Write(bytes)
	if err != nil {
		return fmt.Errorf("failed to write sync snapshot to disk: %s", err)
	}
	return nil
}

func (s *snapshot) loadSnapshot(ctx context.Context) error {
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
	err = json.Unmarshal(bytes, &s.LastModifiedTimes)
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

func (s snapshot) diff(all []git.File) (change diff) {
	currentFilenames := map[string]bool{}
	lastModifiedTimes := s.LastModifiedTimes
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
