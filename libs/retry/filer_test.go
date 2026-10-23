package retry

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flakyFiler struct {
	filer.Filer
	failures int
	err      error
	bodies   []string
}

func (f *flakyFiler) Write(ctx context.Context, path string, r io.Reader, mode ...filer.WriteMode) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.bodies = append(f.bodies, string(b))
	if len(f.bodies) <= f.failures {
		return f.err
	}
	return nil
}

var errTransient = &apierr.APIError{StatusCode: http.StatusGatewayTimeout, Message: "TEMPORARILY_UNAVAILABLE"}

func TestFilerRetriesSeekableReader(t *testing.T) {
	f := &flakyFiler{failures: 1, err: errTransient}
	err := Filer(f).Write(testContext(t), "a", bytes.NewReader([]byte("hello")), filer.OverwriteIfExists)
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "hello"}, f.bodies)
}

func TestFilerRetriesNonSeekableReader(t *testing.T) {
	f := &flakyFiler{failures: 1, err: errTransient}
	// io.NopCloser hides the Seek method.
	r := io.NopCloser(strings.NewReader("hello"))
	err := Filer(f).Write(testContext(t), "a", r, filer.OverwriteIfExists)
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "hello"}, f.bodies)
}

func TestFilerRewindsToInitialReaderPosition(t *testing.T) {
	f := &flakyFiler{failures: 1, err: errTransient}
	r := strings.NewReader("xxhello")
	_, err := r.Seek(2, io.SeekStart)
	require.NoError(t, err)
	err = Filer(f).Write(testContext(t), "a", r, filer.OverwriteIfExists)
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "hello"}, f.bodies)
}

func TestFilerGivesUp(t *testing.T) {
	f := &flakyFiler{failures: 100, err: errTransient}
	err := Filer(f).Write(testContext(t), "a", strings.NewReader("x"), filer.OverwriteIfExists)
	require.Error(t, err)
	assert.Len(t, f.bodies, MaxRetries+1)
}

func TestFilerDoesNotRetryOtherErrors(t *testing.T) {
	f := &flakyFiler{failures: 100, err: &apierr.APIError{StatusCode: http.StatusForbidden}}
	err := Filer(f).Write(testContext(t), "a", strings.NewReader("x"), filer.OverwriteIfExists)
	require.Error(t, err)
	assert.Len(t, f.bodies, 1)
}

func TestFilerDoesNotRetryWithoutOverwrite(t *testing.T) {
	f := &flakyFiler{failures: 100, err: errTransient}
	err := Filer(f).Write(testContext(t), "a", strings.NewReader("x"))
	require.Error(t, err)
	assert.Len(t, f.bodies, 1)
}

// readFiler fails the first `failures` calls of every read-side method with err.
type readFiler struct {
	filer.Filer
	failures int
	err      error
	calls    int
}

func (f *readFiler) next() error {
	f.calls++
	if f.calls <= f.failures {
		return f.err
	}
	return nil
}

func (f *readFiler) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("x")), f.next()
}

func (f *readFiler) ReadDir(ctx context.Context, path string) ([]fs.DirEntry, error) {
	return nil, f.next()
}

func (f *readFiler) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	return nil, f.next()
}

func (f *readFiler) Delete(ctx context.Context, path string, mode ...filer.DeleteMode) error {
	return f.next()
}

func TestFilerRetriesReads(t *testing.T) {
	ctx := testContext(t)
	calls := map[string]func(filer.Filer) error{
		"Read":    func(f filer.Filer) error { _, err := f.Read(ctx, "a"); return err },
		"ReadDir": func(f filer.Filer) error { _, err := f.ReadDir(ctx, "a"); return err },
		"Stat":    func(f filer.Filer) error { _, err := f.Stat(ctx, "a"); return err },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			f := &readFiler{failures: 1, err: errTransient}
			require.NoError(t, call(Filer(f)))
			assert.Equal(t, 2, f.calls)
		})
	}
}

func TestFilerDoesNotRetryNotExist(t *testing.T) {
	f := &readFiler{failures: 1, err: fs.ErrNotExist}
	_, err := Filer(f).Stat(testContext(t), "a")
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.Equal(t, 1, f.calls)
}

func TestFilerDeleteRetries(t *testing.T) {
	f := &readFiler{failures: 1, err: errTransient}
	require.NoError(t, Filer(f).Delete(testContext(t), "a"))
	assert.Equal(t, 2, f.calls)
}

func TestFilerDeleteMissingAfterRetryIsSuccess(t *testing.T) {
	// The first attempt hits a transient error, the retry finds the file gone.
	f := &sequenceFiler{errs: []error{errTransient, fs.ErrNotExist}}
	require.NoError(t, Filer(f).Delete(testContext(t), "a"))
}

func TestFilerDeleteMissingOnFirstAttemptIsError(t *testing.T) {
	f := &sequenceFiler{errs: []error{fs.ErrNotExist}}
	require.ErrorIs(t, Filer(f).Delete(testContext(t), "a"), fs.ErrNotExist)
}

// sequenceFiler returns errs from successive Delete calls.
type sequenceFiler struct {
	filer.Filer
	errs []error
}

func (f *sequenceFiler) Delete(ctx context.Context, path string, mode ...filer.DeleteMode) error {
	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}
