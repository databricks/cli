package filer

import (
	"bytes"
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDbfsClient(t *testing.T) Filer {
	t.Helper()

	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewDbfsClient(client, "/tmp/root")
	require.NoError(t, err)
	return f
}

func TestDbfsClientAgainstTestserver(t *testing.T) {
	ctx := t.Context()
	f := newTestDbfsClient(t)

	// Writing without CreateParentDirectories fails when the parent is missing.
	err := f.Write(ctx, "a/b.txt", strings.NewReader("x"))
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// More than one DBFS block (1 MB), so the write and read are chunked.
	big := bytes.Repeat([]byte("0123456789"), 150_000)
	require.NoError(t, f.Write(ctx, "a/b.txt", bytes.NewReader(big), CreateParentDirectories))

	err = f.Write(ctx, "a/b.txt", strings.NewReader("y"))
	assert.ErrorIs(t, err, fs.ErrExist)

	r, err := f.Read(ctx, "a/b.txt")
	require.NoError(t, err)
	got, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, big, got)

	info, err := f.Stat(ctx, "a/b.txt")
	require.NoError(t, err)
	assert.False(t, info.IsDir())
	assert.Equal(t, int64(len(big)), info.Size())

	info, err = f.Stat(ctx, "a")
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	_, err = f.Stat(ctx, "missing")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	require.NoError(t, f.Mkdir(ctx, "a/sub"))
	entries, err := f.ReadDir(ctx, "a")
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Equal(t, []string{"b.txt", "sub"}, names)

	_, err = f.ReadDir(ctx, "a/b.txt")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	_, err = f.Read(ctx, "a")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	err = f.Delete(ctx, "a")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	require.NoError(t, f.Delete(ctx, "a", DeleteRecursively))
	_, err = f.Stat(ctx, "a")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	err = f.Delete(ctx, "a")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
