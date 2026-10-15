package retry

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/sdk-go/core/apierr"
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

var errTransient = apierr.FromHTTPError(http.StatusGatewayTimeout, nil, []byte(`{"message":"TEMPORARILY_UNAVAILABLE"}`))

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
	f := &flakyFiler{failures: 100, err: apierr.FromHTTPError(http.StatusForbidden, nil, nil)}
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
