package filer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
)

// WorkspaceFiler is the v1 StateFiler backed by the Databricks workspace-files
// API. It wraps libs/filer.Filer and translates between ucm-native types
// (WriteMode, FileInfo, ErrNotFound) and the libs/filer + io/fs equivalents.
//
// Wrapping libs/filer rather than re-implementing the workspace-files
// protocol keeps auth, retry, and SPOG-header handling in a single upstream
// place — on the next upstream sync we inherit fixes for free.
type WorkspaceFiler struct {
	inner libsfiler.Filer
}

// NewWorkspaceFiler constructs a StateFiler that stores files under root
// in the given workspace. root is a workspace path such as
// "/Users/alice@example.com/.ucm/state/dev".
func NewWorkspaceFiler(w *databricks.WorkspaceClient, root string) (StateFiler, error) {
	inner, err := libsfiler.NewWorkspaceFilesClient(w, root)
	if err != nil {
		return nil, fmt.Errorf("ucm filer: init workspace client: %w", err)
	}
	return &WorkspaceFiler{inner: inner}, nil
}

// newWorkspaceFilerFromInner wraps an existing libs/filer.Filer. Exposed at
// package scope so tests can inject a fake.
func newWorkspaceFilerFromInner(inner libsfiler.Filer) *WorkspaceFiler {
	return &WorkspaceFiler{inner: inner}
}

// NewStateFilerFromFiler adapts an arbitrary libs/filer.Filer into a
// StateFiler. Mirrors lock.NewLockerWithFiler: callers that already hold a
// libs/filer.Filer — tests backed by NewLocalClient, or future s3/adls/gcs
// implementations — can reuse it as the state-storage backend without going
// through a workspace client.
func NewStateFilerFromFiler(inner libsfiler.Filer) StateFiler {
	return &WorkspaceFiler{inner: inner}
}

// Read opens the file at path for reading.
func (w *WorkspaceFiler) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	rc, err := w.inner.Read(ctx, path)
	if err != nil {
		return nil, mapErr(path, err)
	}
	return rc, nil
}

// Write writes r to path, translating mode into libs/filer write flags.
func (w *WorkspaceFiler) Write(ctx context.Context, path string, r io.Reader, mode WriteMode) error {
	var flags []libsfiler.WriteMode
	if mode.Has(WriteModeOverwrite) {
		flags = append(flags, libsfiler.OverwriteIfExists)
	}
	if mode.Has(WriteModeCreateParents) {
		flags = append(flags, libsfiler.CreateParentDirectories)
	}
	if err := w.inner.Write(ctx, path, r, flags...); err != nil {
		return mapErr(path, err)
	}
	return nil
}

// Delete removes the file at path. A missing path is treated as success so
// callers don't need to special-case idempotent cleanup.
func (w *WorkspaceFiler) Delete(ctx context.Context, path string) error {
	err := w.inner.Delete(ctx, path)
	if err == nil {
		return nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return mapErr(path, err)
}

// Stat returns metadata for the file at path.
func (w *WorkspaceFiler) Stat(ctx context.Context, path string) (FileInfo, error) {
	info, err := w.inner.Stat(ctx, path)
	if err != nil {
		return nil, mapErr(path, err)
	}
	return fsFileInfo{info}, nil
}

// ReadDir lists the entries of the directory at path.
func (w *WorkspaceFiler) ReadDir(ctx context.Context, path string) ([]FileInfo, error) {
	entries, err := w.inner.ReadDir(ctx, path)
	if err != nil {
		return nil, mapErr(path, err)
	}
	out := make([]FileInfo, 0, len(entries))
	for _, e := range entries {
		info, infoErr := e.Info()
		if infoErr != nil {
			return nil, fmt.Errorf("ucm filer: stat %s: %w", e.Name(), infoErr)
		}
		out = append(out, fsFileInfo{info})
	}
	return out, nil
}

// mapErr translates libs/filer errors into ucm-native sentinel errors.
// fs.ErrNotExist covers both "file does not exist" and "no such directory"
// from libs/filer because both define Is(fs.ErrNotExist) == true.
func mapErr(path string, err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s: %w", ErrNotFound, path, err)
	}
	return err
}

// fsFileInfo adapts an io/fs.FileInfo to the ucm-native FileInfo interface.
type fsFileInfo struct {
	inner fs.FileInfo
}

func (f fsFileInfo) Name() string       { return f.inner.Name() }
func (f fsFileInfo) Size() int64        { return f.inner.Size() }
func (f fsFileInfo) ModTime() time.Time { return f.inner.ModTime() }
func (f fsFileInfo) IsDir() bool        { return f.inner.IsDir() }
