package sync

import (
	"fmt"
	"strings"
	"time"

	"github.com/databricks/bricks/git"
)

type snapshot map[string]time.Time

type diff struct {
	put    []string
	delete []string
}

const SyncSnapshotFile = "repo_snapshot.json"
const BricksDir = ".bricks"

func (s *snapshot) storeSnapshot(root string) error {
	// // create snapshot file
	// configDir := filepath.Join(root, BricksDir)
	// if _, err := os.Stat(configDir); os.IsNotExist(err) {
	// 	err = os.Mkdir(configDir, os.ModeDir|os.ModePerm)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to create config directory: %s", err)
	// 	}
	// }
	// persistedSnapshotPath := filepath.Join(configDir, SyncSnapshotFile)
	// f, err := os.OpenFile(persistedSnapshotPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	// if err != nil {
	// 	return fmt.Errorf("failed to create/open persisted sync snapshot file: %s", err)
	// }
	// defer f.Close()

	// // persist snapshot to disk
	// bytes, err := json.MarshalIndent(s, "", "  ")
	// if err != nil {
	// 	return fmt.Errorf("failed to json marshal in-memory snapshot: %s", err)
	// }
	// _, err = f.Write(bytes)
	// if err != nil {
	// 	return fmt.Errorf("failed to write sync snapshot to disk: %s", err)
	// }
	return nil
}

func (s *snapshot) loadSnapshot(root string) error {
	// persistedSnapshotPath := filepath.Join(root, BricksDir, SyncSnapshotFile)
	// if _, err := os.Stat(persistedSnapshotPath); os.IsNotExist(err) {
	// 	return nil
	// }

	// f, err := os.Open(persistedSnapshotPath)
	// if err != nil {
	// 	return fmt.Errorf("failed to open persisted sync snapshot file: %s", err)
	// }
	// defer f.Close()

	// bytes, err := io.ReadAll(f)
	// if err != nil {
	// 	// clean up these error messages a bit
	// 	return fmt.Errorf("failed to read sync snapshot from disk: %s", err)
	// }
	// err = json.Unmarshal(bytes, s)
	// if err != nil {
	// 	return fmt.Errorf("failed to json unmarshal persisted snapshot: %s", err)
	// }
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
	for _, f := range all {
		// create set of current files to figure out if removals are needed
		currentFilenames[f.Relative] = true
		// get current modified timestamp
		modified := f.Modified()
		lastSeenModified, seen := s[f.Relative]

		if !seen || modified.After(lastSeenModified) {
			change.put = append(change.put, f.Relative)
			s[f.Relative] = modified
		}
	}
	// figure out files in the snapshot, but not on local filesystem
	for relative := range s {
		_, exists := currentFilenames[relative]
		if exists {
			continue
		}
		// add them to a delete batch
		change.delete = append(change.delete, relative)
		// remove the file from snapshot
		delete(s, relative)
	}
	// and remove them from the snapshot
	for _, v := range change.delete {
		delete(s, v)
	}
	return
}
