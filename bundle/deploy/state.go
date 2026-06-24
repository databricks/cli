package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
	"github.com/google/uuid"
)

const (
	DeploymentStateFileName = "deployment.json"
	DeploymentStateVersion  = 1
)

type File struct {
	LocalPath string `json:"local_path"`

	// If true, this file is a notebook.
	// This property must be persisted because notebooks are stripped of their extension.
	// If the local file is no longer present, we need to know what to remove on the workspace side.
	IsNotebook bool `json:"is_notebook"`
}

type Filelist []File

type DeploymentState struct {
	// Version is the version of the deployment state.
	// To be incremented when the schema changes.
	Version int64 `json:"version"`

	// Seq is the sequence number of the deployment state.
	// This number is incremented on every deployment.
	// It is used to detect if the deployment state is stale.
	Seq int64 `json:"seq"`

	// CliVersion is the version of the CLI which created the deployment state.
	CliVersion string `json:"cli_version"`

	// Timestamp is the time when the deployment state was created.
	Timestamp time.Time `json:"timestamp"`

	// Files is a list of files which has been deployed as part of this deployment.
	Files Filelist `json:"files"`

	// UUID uniquely identifying the deployment.
	ID uuid.UUID `json:"id"`
}

// We use this entry type as a proxy to fs.DirEntry.
// When we construct sync snapshot from deployment state,
// we use a fileset.File which embeds fs.DirEntry as the DirEntry field.
// Because we can't marshal/unmarshal fs.DirEntry directly, instead when we unmarshal
// the deployment state, we use this entry type to represent the fs.DirEntry in fileset.File instance.
type entry struct {
	path string
	info fs.FileInfo
}

func newEntry(root vfs.Path, path string) *entry {
	info, err := root.Stat(path)
	if err != nil {
		return &entry{path, nil}
	}

	return &entry{path, info}
}

func (e *entry) Name() string {
	return filepath.Base(e.path)
}

func (e *entry) IsDir() bool {
	// If the entry is nil, it is a non-existent file so return false.
	if e.info == nil {
		return false
	}
	return e.info.IsDir()
}

func (e *entry) Type() fs.FileMode {
	// If the entry is nil, it is a non-existent file so return 0.
	if e.info == nil {
		return 0
	}
	return e.info.Mode()
}

func (e *entry) Info() (fs.FileInfo, error) {
	if e.info == nil {
		return nil, errors.New("no info available")
	}
	return e.info, nil
}

func fromSlice(files []fileset.File) (Filelist, error) {
	var f Filelist
	for k := range files {
		file := &files[k]
		isNotebook, err := file.IsNotebook()
		if err != nil {
			return nil, err
		}
		f = append(f, File{
			LocalPath:  file.Relative,
			IsNotebook: isNotebook,
		})
	}
	return f, nil
}

func (f Filelist) toSlice(root vfs.Path) []fileset.File {
	var files []fileset.File
	for _, file := range f {
		entry := newEntry(root, filepath.ToSlash(file.LocalPath))

		// Snapshots created with versions <= v0.220.0 use platform-specific
		// paths (i.e. with backslashes). Files returned by [libs/fileset] always
		// contain forward slashes after this version. Normalize before using.
		relative := filepath.ToSlash(file.LocalPath)
		if file.IsNotebook {
			files = append(files, fileset.NewNotebookFile(root, entry, relative))
		} else {
			files = append(files, fileset.NewSourceFile(root, entry, relative))
		}
	}
	return files
}

func isLocalStateStale(local, remote io.Reader) bool {
	localState, err := loadState(local)
	if err != nil {
		return true
	}

	remoteState, err := loadState(remote)
	if err != nil {
		return false
	}

	return localState.Seq < remoteState.Seq
}

func validateRemoteStateCompatibility(remote io.Reader) error {
	state, err := loadState(remote)
	if err != nil {
		return err
	}

	// If the remote state version is greater than the CLI version, we can't proceed.
	if state.Version > DeploymentStateVersion {
		return fmt.Errorf("remote deployment state is incompatible with the current version of the CLI, please upgrade to at least %s", state.CliVersion)
	}

	return nil
}

func loadState(r io.Reader) (*DeploymentState, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var s DeploymentState
	err = json.Unmarshal(content, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func getPathToStateFile(ctx context.Context, b *bundle.Bundle) (string, error) {
	cacheDir, err := b.LocalStateDir(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot get bundle cache directory: %w", err)
	}
	return filepath.Join(cacheDir, DeploymentStateFileName), nil
}
