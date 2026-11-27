package filer

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dbfsServer serves the DBFS API from the test server's fake workspace. The
// first dropAddBlock add-block calls and the first dropClose close calls find
// their stream handle gone, like the real backend does when it loses a stream.
type dbfsServer struct {
	mu           sync.Mutex
	dropAddBlock int
	dropClose    int
	creates      []string
}

// dropHandle closes the request's stream on the fake and then runs handler, which
// fails with the backend's "No such stream with handle" error. Closing commits
// the partial file, so a restarted upload without overwrite would conflict with it.
func dropHandle(req testserver.Request, handler func(testserver.Request) testserver.Response) testserver.Response {
	req.Workspace.DbfsClose(req)
	return handler(req)
}

func newDbfsClientWithServer(t *testing.T, d *dbfsServer) *DbfsClient {
	t.Helper()
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/dbfs/get-status", func(req testserver.Request) any {
		return req.Workspace.DbfsGetStatus(req)
	})
	server.Handle("POST", "/api/2.0/dbfs/mkdirs", func(req testserver.Request) any {
		return req.Workspace.DbfsMkdirs(req)
	})
	server.Handle("GET", "/api/2.0/dbfs/read", func(req testserver.Request) any {
		return req.Workspace.DbfsRead(req)
	})
	server.Handle("POST", "/api/2.0/dbfs/create", func(req testserver.Request) any {
		d.mu.Lock()
		d.creates = append(d.creates, string(req.Body))
		d.mu.Unlock()
		return req.Workspace.DbfsCreate(req)
	})
	server.Handle("POST", "/api/2.0/dbfs/add-block", func(req testserver.Request) any {
		d.mu.Lock()
		drop := d.dropAddBlock > 0
		d.dropAddBlock--
		d.mu.Unlock()
		if drop {
			return dropHandle(req, req.Workspace.DbfsAddBlock)
		}
		return req.Workspace.DbfsAddBlock(req)
	})
	server.Handle("POST", "/api/2.0/dbfs/close", func(req testserver.Request) any {
		d.mu.Lock()
		drop := d.dropClose > 0
		d.dropClose--
		d.mu.Unlock()
		if drop {
			return dropHandle(req, req.Workspace.DbfsClose)
		}
		return req.Workspace.DbfsClose(req)
	})

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewDbfsClient(client, "/tmp/root")
	require.NoError(t, err)
	return f.(*DbfsClient)
}

func readDbfsFile(t *testing.T, f *DbfsClient, name string) string {
	t.Helper()
	r, err := f.Read(t.Context(), name)
	require.NoError(t, err)
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(b)
}

func TestDbfsClientRestartsUploadWhenAddBlockLosesHandle(t *testing.T) {
	d := &dbfsServer{dropAddBlock: 1}
	f := newDbfsClientWithServer(t, d)

	// The source is a non-seekable stream and the write does not overwrite: the
	// restart must replay the buffered bytes and replace its own partial file.
	err := f.Write(t.Context(), "hello", onlyReader{strings.NewReader("abc")}, CreateParentDirectories)
	require.NoError(t, err)
	assert.Equal(t, "abc", readDbfsFile(t, f, "hello"))
	assert.Equal(t, []string{
		`{"path":"/tmp/root/hello"}`,
		`{"overwrite":true,"path":"/tmp/root/hello"}`,
	}, d.creates)
}

func TestDbfsClientRestartsUploadWhenCloseLosesHandle(t *testing.T) {
	d := &dbfsServer{dropClose: 1}
	f := newDbfsClientWithServer(t, d)

	err := f.Write(t.Context(), "hello", onlyReader{strings.NewReader("abc")}, CreateParentDirectories)
	require.NoError(t, err)
	assert.Equal(t, "abc", readDbfsFile(t, f, "hello"))
	assert.Len(t, d.creates, 2)
}

func TestDbfsClientRestartsSeekableUploadFromStartOffset(t *testing.T) {
	d := &dbfsServer{dropAddBlock: 1}
	f := newDbfsClientWithServer(t, d)

	r := bytes.NewReader([]byte("xxabc"))
	_, err := r.Seek(2, io.SeekStart)
	require.NoError(t, err)

	// The seekable source is not buffered, even when it exceeds the buffer size.
	setDbfsReplayBufferSize(t, 1)
	err = f.Write(t.Context(), "hello", r, CreateParentDirectories, OverwriteIfExists)
	require.NoError(t, err)
	assert.Equal(t, "abc", readDbfsFile(t, f, "hello"))
	assert.Len(t, d.creates, 2)
}

func TestDbfsClientRestartsUploadOnlyOnce(t *testing.T) {
	d := &dbfsServer{dropAddBlock: 2}
	f := newDbfsClientWithServer(t, d)

	err := f.Write(t.Context(), "hello", onlyReader{strings.NewReader("abc")}, CreateParentDirectories)
	assert.ErrorContains(t, err, "No such stream with handle")
	assert.Len(t, d.creates, 2)
}

func TestDbfsClientDoesNotRestartLargeNonSeekableUpload(t *testing.T) {
	d := &dbfsServer{dropAddBlock: 1}
	f := newDbfsClientWithServer(t, d)
	setDbfsReplayBufferSize(t, 2)

	err := f.Write(t.Context(), "hello", onlyReader{strings.NewReader("abc")}, CreateParentDirectories)
	assert.ErrorContains(t, err, "No such stream with handle")
	assert.Len(t, d.creates, 1)
}

func TestDbfsClientRestartsLargeNonSeekableUploadThatFailsEarly(t *testing.T) {
	d := &dbfsServer{dropAddBlock: 1}
	f := newDbfsClientWithServer(t, d)

	// The first add-block fails after one 1 MiB block was read: the buffered
	// block is replayed and the rest of the source is streamed after it.
	content := strings.Repeat("a", 1024*1024) + strings.Repeat("b", 1024*1024) + "c"
	setDbfsReplayBufferSize(t, 1024*1024+10)
	err := f.Write(t.Context(), "hello", onlyReader{strings.NewReader(content)}, CreateParentDirectories)
	require.NoError(t, err)
	assert.Equal(t, content, readDbfsFile(t, f, "hello"))
	assert.Len(t, d.creates, 2)
}

func TestDbfsClientRestartsNonSeekableUploadOfBufferSize(t *testing.T) {
	d := &dbfsServer{dropAddBlock: 1}
	f := newDbfsClientWithServer(t, d)
	setDbfsReplayBufferSize(t, 3)

	err := f.Write(t.Context(), "hello", onlyReader{strings.NewReader("abc")}, CreateParentDirectories)
	require.NoError(t, err)
	assert.Equal(t, "abc", readDbfsFile(t, f, "hello"))
}

func TestDbfsClientDoesNotRestartOnOtherErrors(t *testing.T) {
	d := &dbfsServer{}
	f := newDbfsClientWithServer(t, d)

	require.NoError(t, f.Write(t.Context(), "hello", strings.NewReader("abc"), CreateParentDirectories))
	err := f.Write(t.Context(), "hello", strings.NewReader("def"), CreateParentDirectories)
	assert.ErrorIs(t, err, fileAlreadyExistsError{"/tmp/root/hello"})
	assert.Len(t, d.creates, 2)
	assert.Equal(t, "abc", readDbfsFile(t, f, "hello"))
}

func setDbfsReplayBufferSize(t *testing.T, n int) {
	old := dbfsReplayBufferSize
	dbfsReplayBufferSize = n
	t.Cleanup(func() { dbfsReplayBufferSize = old })
}
