package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/fileset"
)

const DeploymentStateFileName = "deployment.json"

type File struct {
	Path string `json:"path"`
}

type Filelist []File

type DeploymentState struct {
	Version   string    `json:"version"`
	Seq       int64     `json:"seq"`
	Timestamp time.Time `json:"timestamp"`
	Files     Filelist  `json:"files"`
}

// We use this entry type as a proxy to fs.DirEntry.
// When we construct sync snapshot from deployment state,
// we use a fileset.File which embeds we use fs.DirEntry as the DirEntry field.
// Because we can't marshal/unmarshal fs.DirEntry directly, instead when we unmarshal
// the deployment state, we use this entry type to represent the fs.DirEntry in fileset.File instance.
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
			Path: file.Relative,
		})
	}
	return f
}

func (f Filelist) ToSlice(basePath string) []fileset.File {
	var files []fileset.File
	for _, file := range f {
		absPath := filepath.Join(basePath, file.Path)
		files = append(files, fileset.File{
			DirEntry: newEntry(absPath),
			Absolute: absPath,
			Relative: file.Path,
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

	return localState.Seq < remoteState.Seq
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
