package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/fileset"
)

const DeploymentStateFileName = "deployment-state.json"

type File struct {
	Absolute string `json:"absolute"`
	Relative string `json:"relative"`
}

type Filelist []File

type DeploymentState struct {
	Version int64    `json:"version"`
	Files   Filelist `json:"files"`
}

type entry struct {
	path string
	info fs.FileInfo
}

func newEntry(path string) *entry {
	info, err := os.Stat(path)
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
		return nil, fmt.Errorf("no info available")
	}
	return e.info, nil
}

func FromSlice(files []fileset.File) Filelist {
	var f Filelist
	for _, file := range files {
		f = append(f, File{
			Absolute: file.Absolute,
			Relative: file.Relative,
		})
	}
	return f
}

func (f Filelist) ToSlice() []fileset.File {
	var files []fileset.File
	for _, file := range f {
		files = append(files, fileset.File{
			DirEntry: newEntry(file.Absolute),
			Absolute: file.Absolute,
			Relative: file.Relative,
		})
	}
	return files
}

func isLocalStateStale(local io.Reader, remote io.Reader) bool {
	localState, err := loadState(local)
	if err != nil {
		return true
	}

	remoteState, err := loadState(remote)
	if err != nil {
		return false
	}

	return localState.Version < remoteState.Version
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
	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot get bundle cache directory: %w", err)
	}
	return filepath.Join(cacheDir, DeploymentStateFileName), nil
}
