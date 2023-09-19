package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/fileset"
)

type SyncFiles struct {
	// Map of all files present in the remote repo with the:
	// key: relative file path from project root
	// value: last time the remote instance of this file was updated
	LastModifiedTimes map[string]time.Time `json:"last_modified_times"`

	// This map maps local file names to their remote names
	// eg. notebook named "foo.py" locally would be stored as "foo", thus this
	// map will contain an entry "foo.py" -> "foo"
	LocalToRemoteNames map[string]string `json:"local_to_remote_names"`

	// Inverse of localToRemoteNames. Together the form a bijective mapping (ie
	// there is a 1:1 unique mapping between local and remote name)
	RemoteToLocalNames map[string]string `json:"remote_to_local_names"`
}

// TODO: add error logging here to help determine what went wrong.
func newSyncFiles(ctx context.Context, localFiles []fileset.File) (*SyncFiles, error) {
	sf := &SyncFiles{
		LastModifiedTimes:  make(map[string]time.Time),
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}

	// Validate no duplicate entries exist for the same file.
	seen := make(map[string]fileset.File)
	for _, f := range localFiles {
		if _, ok := seen[f.Relative]; !ok {
			seen[f.Relative] = f
		} else {
			// TODO: Log this using our error logger instead.
			return nil, fmt.Errorf("error while computing sync snapshot. Two fileset pointers passed for local file %s, %#v and %#v", f.Relative, seen[f.Relative], f)
		}
	}

	// Compute the new files state for the snapshot
	for _, f := range localFiles {
		sf.LastModifiedTimes[f.Relative] = f.Modified()
		remoteName, err := f.RemotePath()
		if err != nil {
			return nil, err
		}

		if existingLocalName, ok := sf.RemoteToLocalNames[remoteName]; ok {
			return nil, fmt.Errorf("both %s and %s point to the same remote file location %s. Please remove one of them from your local project", existingLocalName, f.Relative, remoteName)
		}

		sf.LocalToRemoteNames[f.Relative] = remoteName
		sf.RemoteToLocalNames[remoteName] = f.Relative
	}
	// TODO: wrap error from validate here, to know source is local computation.
	// Do the same fore remote read.
	if err := sf.validate(); err != nil {
		return nil, fmt.Errorf("error while computing sync files state from local files. Please report this on https://github.com/databricks/cli: %w", err)
	}
	return sf, nil
}

// This function validates the syncFiles is constant with respect to invariants
// we expect here, namely:
//  1. All files with an entry in "LastModifiedTimes" also have an entry in "LocalToRemoteNames"
//     and vice versa
//  2. "LocalToRemoteNames" and "RemoteToLocalNames" together form an one to one mapping
//     of local <-> remote file names.
func (sf *SyncFiles) validate() error {
	// Validate condition (1), ie "LastModifiedTimes" and "LocalToRemoteNames"
	// have the same underlying set of local files as keys.
	for k, _ := range sf.LastModifiedTimes {
		if _, ok := sf.LocalToRemoteNames[k]; !ok {
			return fmt.Errorf("local file path %s is missing an entry for the corresponding remote file path in sync state", k)
		}
	}
	for k, _ := range sf.LocalToRemoteNames {
		if _, ok := sf.LastModifiedTimes[k]; !ok {
			return fmt.Errorf("local file path %s is missing an entry for it's last modified time in sync state", k)
		}
	}

	// Validate condition (2), ie "LocalToRemoteNames" and "RemoteToLocalNames" form
	// a 1:1 mapping.
	for localName, remoteName := range sf.LocalToRemoteNames {
		if _, ok := sf.RemoteToLocalNames[remoteName]; !ok {
			return fmt.Errorf("remote file path %s is missing an entry for the corresponding local file path in sync state", remoteName)
		}
		if sf.RemoteToLocalNames[remoteName] != localName {
			return fmt.Errorf("inconsistent mapping of local <-> remote file paths. Local file %s points to %s. Remote file %s points to %s", localName, remoteName, remoteName, sf.RemoteToLocalNames[localName])
		}
	}
	for remoteName, localName := range sf.RemoteToLocalNames {
		if _, ok := sf.LocalToRemoteNames[localName]; !ok {
			return fmt.Errorf("local file path %s is missing an entry for the corresponding remote file path in sync state", localName)
		}
		if sf.RemoteToLocalNames[remoteName] != localName {
			return fmt.Errorf("inconsistent mapping of local <-> remote file paths. Remote file %s points to %s. Local file %s points to %s", remoteName, localName, localName, sf.LocalToRemoteNames[localName])
		}
	}
	return nil
}
