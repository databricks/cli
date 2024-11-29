package template

import (
	"bytes"
	"context"
	"io/fs"
	"slices"

	"github.com/databricks/cli/libs/filer"
)

// Interface representing a file to be materialized from a template into a project
// instance
type file interface {
	// Path of the file relative to the root of the instantiated template.
	// This is where the file is written to when persisting the template to disk.
	// Must be slash-separated.
	RelPath() string

	// Write file to disk at the destination path.
	Write(ctx context.Context, out filer.Filer) error

	// contents returns the file contents as a byte slice.
	// This is used for testing purposes.
	contents() ([]byte, error)
}

type copyFile struct {
	// Permissions bits for the destination file
	perm fs.FileMode

	// Destination path for the file.
	relPath string

	// [fs.FS] rooted at template root. Used to read srcPath.
	srcFS fs.FS

	// Relative path from template root for file to be copied.
	srcPath string
}

func (f *copyFile) RelPath() string {
	return f.relPath
}

func (f *copyFile) Write(ctx context.Context, out filer.Filer) error {
	src, err := f.srcFS.Open(f.srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	return out.Write(ctx, f.relPath, src, filer.CreateParentDirectories, filer.WriteMode(f.perm))
}

func (f *copyFile) contents() ([]byte, error) {
	return fs.ReadFile(f.srcFS, f.srcPath)
}

type inMemoryFile struct {
	// Permissions bits for the destination file
	perm fs.FileMode

	// Destination path for the file.
	relPath string

	// Contents of the file.
	content []byte
}

func (f *inMemoryFile) RelPath() string {
	return f.relPath
}

func (f *inMemoryFile) Write(ctx context.Context, out filer.Filer) error {
	return out.Write(ctx, f.relPath, bytes.NewReader(f.content), filer.CreateParentDirectories, filer.WriteMode(f.perm))
}

func (f *inMemoryFile) contents() ([]byte, error) {
	return slices.Clone(f.content), nil
}
