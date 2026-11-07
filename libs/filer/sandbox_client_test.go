package filer_test

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSandboxTestClient(t *testing.T, server *testserver.Server) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
		// The SDK's default rate limit would throttle the concurrent writes below.
		RateLimitPerSecond: 10000,
	})
	require.NoError(t, err)
	return w
}

// newSandboxTestFiler returns a sandbox filer rooted at "/tmp", backed by the testserver
// fake of the sandbox file APIs, which reproduces the server semantics the filer works around.
func newSandboxTestFiler(t *testing.T) filer.Filer {
	t.Helper()
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	f, err := filer.NewSandboxClient(newSandboxTestClient(t, server), testserver.TestDefaultSandboxId, "/tmp")
	require.NoError(t, err)
	return f
}

func readString(t *testing.T, f filer.Filer, name string) string {
	t.Helper()
	r, err := f.Read(t.Context(), name)
	require.NoError(t, err)
	defer r.Close()
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(b)
}

func TestSandboxClientRejectsInvalidArguments(t *testing.T) {
	w := &databricks.WorkspaceClient{}
	for _, id := range []string{"", ".", "..", "a/b"} {
		_, err := filer.NewSandboxClient(w, id, "/")
		assert.ErrorContains(t, err, "invalid sandbox ID")
	}
	_, err := filer.NewSandboxClient(w, "id", "relative")
	assert.ErrorContains(t, err, "sandbox root must be absolute")
}

func TestSandboxClientWriteDoesNotOverwrite(t *testing.T) {
	// The upload API always replaces, so exclusive writes go through a temporary file
	// and a rename, which must not leave the temporary file behind.
	ctx := t.Context()
	f := newSandboxTestFiler(t)

	require.NoError(t, f.Write(ctx, "file", strings.NewReader("one")))
	assert.ErrorIs(t, f.Write(ctx, "file", strings.NewReader("two")), fs.ErrExist)
	assert.Equal(t, "one", readString(t, f, "file"))

	entries, err := f.ReadDir(ctx, ".")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "file", entries[0].Name())
}

func TestSandboxClientConcurrentExclusiveWrite(t *testing.T) {
	ctx := t.Context()
	f := newSandboxTestFiler(t)

	const writers = 8
	var wg sync.WaitGroup
	errs := make([]error, writers)
	for i := range writers {
		wg.Go(func() {
			errs[i] = f.Write(ctx, "file", strings.NewReader(fmt.Sprintf("writer-%d", i)))
		})
	}
	wg.Wait()

	winner := -1
	for i, err := range errs {
		if err == nil {
			assert.Equal(t, -1, winner, "more than one writer created the file")
			winner = i
			continue
		}
		assert.ErrorIs(t, err, fs.ErrExist)
	}
	require.NotEqual(t, -1, winner, "no writer created the file")
	assert.Equal(t, fmt.Sprintf("writer-%d", winner), readString(t, f, "file"))
}

func TestSandboxClientWriteMissingParent(t *testing.T) {
	// The upload API always creates missing parents.
	err := newSandboxTestFiler(t).Write(t.Context(), "missing/file", strings.NewReader("x"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestSandboxClientDeleteMissing(t *testing.T) {
	// The delete API succeeds if the path is absent.
	err := newSandboxTestFiler(t).Delete(t.Context(), "missing")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestSandboxClientGenericErrors(t *testing.T) {
	// The server reports each of these with a generic status, which the filer
	// disambiguates with a stat.
	ctx := t.Context()
	f := newSandboxTestFiler(t)
	require.NoError(t, f.Write(ctx, "dir/file", strings.NewReader("x"), filer.CreateParentDirectories))

	_, err := f.Read(ctx, "dir")
	assert.ErrorIs(t, err, fs.ErrInvalid, "read of a directory")
	_, err = f.ReadDir(ctx, "dir/file")
	assert.ErrorIs(t, err, fs.ErrInvalid, "list of a file")
	assert.ErrorIs(t, f.Delete(ctx, "dir"), fs.ErrInvalid, "delete of a non-empty directory")
	assert.ErrorIs(t, f.Mkdir(ctx, "dir/file"), fs.ErrExist, "mkdir onto a file")
	err = f.Write(ctx, "dir", strings.NewReader("x"), filer.OverwriteIfExists)
	assert.ErrorIs(t, err, fs.ErrInvalid, "overwrite of a directory")
}

func TestSandboxClientPermissionDenied(t *testing.T) {
	// The fake doesn't model filesystem permissions, so stub the response a real
	// sandbox returns for a path the sandbox user can't access.
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/sandbox-files/sandboxes/{sandbox_id}/paths:describe", func(req testserver.Request) any {
		return testserver.Response{StatusCode: 403, Body: map[string]string{
			"error_code": "PERMISSION_DENIED", "message": "Access to the requested operation is denied.",
		}}
	})
	f, err := filer.NewSandboxClient(newSandboxTestClient(t, server), testserver.TestDefaultSandboxId, "/")
	require.NoError(t, err)

	_, err = f.Stat(t.Context(), "/root/x")
	assert.ErrorIs(t, err, fs.ErrPermission)
}
