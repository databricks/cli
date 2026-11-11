package retry

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"slices"

	"github.com/databricks/cli/libs/filer"
)

// retryingFiler retries calls that fail with a transient server error and are safe to repeat.
type retryingFiler struct {
	filer.Filer
}

// Filer wraps f so that Write, Read, ReadDir, Stat and Delete retry on transient errors.
// Only writes with [filer.OverwriteIfExists] are retried: they are idempotent, whereas repeating
// a write that does not overwrite could fail on the file its own earlier attempt created.
// A retried Delete treats a missing file as success, since the earlier attempt may have removed it.
// Mkdir is passed through.
func Filer(f filer.Filer) filer.Filer {
	return &retryingFiler{Filer: f}
}

func (r *retryingFiler) Write(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) error {
	if !slices.Contains(mode, filer.OverwriteIfExists) {
		return r.Filer.Write(ctx, path, reader, mode...)
	}

	// Every attempt needs the full contents, so rewind the reader or buffer it.
	rewind := func() (io.Reader, error) { return reader, nil }
	if seeker, ok := reader.(io.ReadSeeker); ok {
		start, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		rewind = func() (io.Reader, error) {
			_, err := seeker.Seek(start, io.SeekStart)
			return reader, err
		}
	} else {
		body, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		rewind = func() (io.Reader, error) { return bytes.NewReader(body), nil }
	}

	_, err := With(ctx, IsTransient, func() (struct{}, error) {
		attempt, err := rewind()
		if err != nil {
			return struct{}{}, err
		}
		return struct{}{}, r.Filer.Write(ctx, path, attempt, mode...)
	})
	return err
}

func (r *retryingFiler) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return With(ctx, IsTransientRead, func() (io.ReadCloser, error) {
		return r.Filer.Read(ctx, path)
	})
}

func (r *retryingFiler) ReadDir(ctx context.Context, path string) ([]fs.DirEntry, error) {
	return With(ctx, IsTransientRead, func() ([]fs.DirEntry, error) {
		return r.Filer.ReadDir(ctx, path)
	})
}

func (r *retryingFiler) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	return With(ctx, IsTransientRead, func() (fs.FileInfo, error) {
		return r.Filer.Stat(ctx, path)
	})
}

func (r *retryingFiler) Delete(ctx context.Context, path string, mode ...filer.DeleteMode) error {
	retried := false
	_, err := With(ctx, IsTransient, func() (struct{}, error) {
		err := r.Filer.Delete(ctx, path, mode...)
		if retried && errors.Is(err, fs.ErrNotExist) {
			return struct{}{}, nil
		}
		retried = true
		return struct{}{}, err
	})
	return err
}
