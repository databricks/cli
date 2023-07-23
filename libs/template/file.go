package template

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/libs/filer"
)

// Interface for an in memory representation of a file
type file interface {
	// Path file will be persisted at, if PersistToDisk is called
	Path() string

	// Does the path of the file, relative to the project root match any of the
	// specified skip patterns
	IsSkipped(patterns []string) (bool, error)

	// This function writes this file onto the disk
	PersistToDisk() error
}

type fileCommon struct {
	// Root path for the project instance. This path uses the system's default
	// file separator. For example /foo/bar on Unix and C:\foo\bar on windows
	root string

	// Unix like relPath for the file (using '/' as the separator). This path
	// is relative to the root. Using unix like relative paths enables skip patterns
	// to work across both windows and unix based operating systems.
	relPath string

	// Permissions bits for the file
	perm fs.FileMode
}

func (f *fileCommon) Path() string {
	return filepath.Join(f.root, filepath.FromSlash(f.relPath))
}

func (f *fileCommon) IsSkipped(patterns []string) (bool, error) {
	for _, pattern := range patterns {
		isMatch, err := path.Match(pattern, f.relPath)
		if err != nil {
			return false, err
		}
		if isMatch {
			return true, nil
		}
	}
	return false, nil
}

type copyFile struct {
	*fileCommon

	ctx context.Context

	// Path of the source file that should be copied over.
	srcPath string

	// Filer to use to read source path
	srcFiler filer.Filer
}

func (f *copyFile) PersistToDisk() error {
	path := f.Path()
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	srcFile, err := f.srcFiler.Read(f.ctx, f.srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, f.perm)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return err
}

type inMemoryFile struct {
	*fileCommon

	content []byte
}

func (f *inMemoryFile) PersistToDisk() error {
	path := f.Path()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(path, f.content, f.perm)
}
