package sync

import (
	"fmt"
	"strings"
	"time"

	"github.com/databricks/bricks/git"
)

// TODO: persist snapshot locally / or fetch remote snapshot and see the drift
// See how dbx sync handles it
// Do we plan to store this in a local file system ?
// We should not have to reupload the files on reboots
type snapshot map[string]time.Time

type diff struct {
	put    []string
	delete []string
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
		// and add them to a delete batch
		change.delete = append(change.delete, relative)
	}
	// and remove them from the snapshot
	for _, v := range change.delete {
		delete(s, v)
	}
	return
}
