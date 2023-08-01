package template

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/filer"
)

// Interface representing a file to be materialized from a template into a project
// instance
type file interface {
	// Destination path for file. This is where the file will be created when
	// PersistToDisk is called.
	DstPath() *destinationPath

	// Write file to disk at the destination path.
	PersistToDisk() error
}

type destinationPath struct {
	// Root path for the project instance. This path uses the system's default
	// file separator. For example /foo/bar on Unix and C:\foo\bar on windows
	root string

	// Unix like file path relative to the "root" of the instantiated project. Is used to
	// evaluate whether the file should be skipped by comparing it to a list of
	// skip glob patterns.
	relPath string
}

// Absolute path of the file, in the os native format. For example /foo/bar on
// Unix and C:\foo\bar on windows
func (f *destinationPath) absPath() string {
	return filepath.Join(f.root, filepath.FromSlash(f.relPath))
}

type copyFile struct {
	ctx context.Context

	// Permissions bits for the destination file
	perm fs.FileMode

	dstPath *destinationPath

	// Filer rooted at template root. Used to read srcPath.
	srcFiler filer.Filer

	// Relative path from template root for file to be copied.
	srcPath string
}

func (f *copyFile) DstPath() *destinationPath {
	return f.dstPath
}

func (f *copyFile) PersistToDisk() error {
	path := f.DstPath().absPath()
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
	dstPath *destinationPath

	content []byte

	// Permissions bits for the destination file
	perm fs.FileMode
}

func (f *inMemoryFile) DstPath() *destinationPath {
	return f.dstPath
}

func (f *inMemoryFile) PersistToDisk() error {
	path := f.DstPath().absPath()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	return os.WriteFile(path, f.content, f.perm)
}
