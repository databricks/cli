package retry

import (
	"bytes"
	"context"
	"io"
	"slices"

	"github.com/databricks/cli/libs/filer"
)

// retryingFiler retries overwriting writes that fail with a transient server error.
type retryingFiler struct {
	filer.Filer
}

// Filer wraps f so that Write retries on transient server errors.
// Only writes with [filer.OverwriteIfExists] are retried: they are idempotent, whereas repeating
// a write that does not overwrite could fail on the file its own earlier attempt created.
// All other methods are passed through.
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
