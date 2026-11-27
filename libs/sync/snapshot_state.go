package sync

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/notebook"
)

// SnapshotState keeps track of files on the local filesystem and their corresponding
// entries in WSFS.
type SnapshotState struct { //nolint:recvcheck // value receiver for ToSlash() copy, pointer for mutation
	// Map of local file names to their last recorded modified time. Files found
	// to have a newer mtime have their content synced to their remote version.
	LastModifiedTimes map[string]time.Time `json:"last_modified_times"`

	// Map of local file names to corresponding remote names.
	// For example: A notebook named "foo.py" locally would be stored as "foo"
	// in WSFS, and the entry would be: {"foo.py": "foo"}
	LocalToRemoteNames map[string]string `json:"local_to_remote_names"`

	// Inverse of LocalToRemoteNames. Together they form a 1:1 mapping where all
	// the remote names and local names are unique.
	RemoteToLocalNames map[string]string `json:"remote_to_local_names"`
}

// Convert an array of files on the local file system to a SnapshotState representation.
func NewSnapshotState(localFiles []fileset.File) (*SnapshotState, error) {
	state, _, err := newSnapshotStateFrom(localFiles, nil)
	return state, err
}

// newSnapshotStateFrom is [NewSnapshotState] for a sync that last left the
// workspace in the previous state (nil if there is none). A file whose notebook
// type cannot be determined, such as a .ipynb with invalid JSON, keeps its remote
// name from previous: dropping it would make the diff delete its remote copy.
// Its mtime is reset, so it is uploaded once it can be read again, whatever its
// mtime then is. reset reports whether that changed any mtime from previous.
func newSnapshotStateFrom(localFiles []fileset.File, previous *SnapshotState) (state *SnapshotState, reset bool, err error) {
	fs := &SnapshotState{
		LastModifiedTimes:  make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}

	// Expect no files to have a duplicate entry in the input array.
	seen := make(map[string]fileset.File)
	for _, f := range localFiles {
		if _, ok := seen[f.Relative]; !ok {
			seen[f.Relative] = f
		} else {
			return nil, false, fmt.Errorf("expected only one entry per file. Found duplicate entries for file: %s", f.Relative)
		}
	}

	// Compute the new state.
	for k := range localFiles {
		f := &localFiles[k]
		// Compute the remote name the file will have in WSFS
		remoteName := f.Relative
		modified := f.Modified()
		isNotebook, err := f.IsNotebook()
		if err != nil {
			// Trying to upload such a file to the workspace would fail, so don't.
			// Keeping its previous remote name leaves the remote copy as it is
			// until the file can be read again. A file removed since it was
			// listed is gone, so its remote copy is deleted as usual.
			previousRemoteName, ok := previous.remoteName(f.Relative)
			if !ok || errors.Is(err, iofs.ErrNotExist) {
				continue
			}
			remoteName = previousRemoteName
			modified = time.Unix(0, 0)
			if !previous.LastModifiedTimes[f.Relative].Equal(modified) {
				reset = true
			}
		} else if isNotebook {
			remoteName = notebook.StripExtension(remoteName)
		}

		// Add the file to snapshot state
		fs.LastModifiedTimes[f.Relative] = modified
		if existingLocalName, ok := fs.RemoteToLocalNames[remoteName]; ok {
			return nil, false, fmt.Errorf("both %s and %s point to the same remote file location %s. Please remove one of them from your local project", existingLocalName, f.Relative, remoteName)
		}

		fs.LocalToRemoteNames[f.Relative] = remoteName
		fs.RemoteToLocalNames[remoteName] = f.Relative
	}
	return fs, reset, nil
}

// remoteName returns the remote name of a tracked local file. Safe on a nil state.
func (fs *SnapshotState) remoteName(localName string) (string, bool) {
	if fs == nil {
		return "", false
	}
	name, ok := fs.LocalToRemoteNames[localName]
	return name, ok
}

func (fs *SnapshotState) ResetLastModifiedTimes() {
	for k := range fs.LastModifiedTimes {
		fs.LastModifiedTimes[k] = time.Unix(0, 0)
	}
}

// Consistency checks for the sync files state representation. These are invariants
// that downstream code for computing changes to apply to WSFS depends on.
//
// Invariants:
//  1. All entries in LastModifiedTimes have a corresponding entry in LocalToRemoteNames
//     and vice versa.
//  2. LocalToRemoteNames and RemoteToLocalNames together form a 1:1 mapping of
//     local <-> remote file names.
func (fs *SnapshotState) validate() error {
	// Validate invariant (1)
	for localName := range fs.LastModifiedTimes {
		if _, ok := fs.LocalToRemoteNames[localName]; !ok {
			return fmt.Errorf("invalid sync state representation. Local file %s is missing the corresponding remote file", localName)
		}
	}
	for localName := range fs.LocalToRemoteNames {
		if _, ok := fs.LastModifiedTimes[localName]; !ok {
			return fmt.Errorf("invalid sync state representation. Local file %s is missing it's last modified time", localName)
		}
	}

	// Validate invariant (2)
	for localName, remoteName := range fs.LocalToRemoteNames {
		if _, ok := fs.RemoteToLocalNames[remoteName]; !ok {
			return fmt.Errorf("invalid sync state representation. Remote file %s is missing the corresponding local file", remoteName)
		}
		if fs.RemoteToLocalNames[remoteName] != localName {
			return fmt.Errorf("invalid sync state representation. Inconsistent values found. Local file %s points to %s. Remote file %s points to %s", localName, remoteName, remoteName, fs.RemoteToLocalNames[remoteName])
		}
	}
	for remoteName, localName := range fs.RemoteToLocalNames {
		if _, ok := fs.LocalToRemoteNames[localName]; !ok {
			return fmt.Errorf("invalid sync state representation. local file %s is missing the corresponding remote file", localName)
		}
		if fs.LocalToRemoteNames[localName] != remoteName {
			return fmt.Errorf("invalid sync state representation. Inconsistent values found. Remote file %s points to %s. Local file %s points to %s", remoteName, localName, localName, fs.LocalToRemoteNames[localName])
		}
	}
	return nil
}

// ToSlash ensures all local paths in the snapshot state
// are slash-separated. Returns a new snapshot state.
func (fs SnapshotState) ToSlash() *SnapshotState {
	new := SnapshotState{
		LastModifiedTimes:  make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}

	// Keys are local paths.
	for k, v := range fs.LastModifiedTimes {
		new.LastModifiedTimes[filepath.ToSlash(k)] = v
	}

	// Keys are local paths.
	for k, v := range fs.LocalToRemoteNames {
		new.LocalToRemoteNames[filepath.ToSlash(k)] = v
	}

	// Values are remote paths.
	for k, v := range fs.RemoteToLocalNames {
		new.RemoteToLocalNames[k] = filepath.ToSlash(v)
	}

	return &new
}
