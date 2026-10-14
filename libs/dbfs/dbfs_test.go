package dbfs

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDBFS stores files written via create/add-block/close and serves reads.
type fakeDBFS struct {
	files   map[string][]byte
	handles map[int64]string
	next    int64
}

func (f *fakeDBFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if r.Method == http.MethodPost {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	q := r.URL.Query()
	reply := func(v any) { _ = json.NewEncoder(w).Encode(v) }
	switch r.URL.Path {
	case "/api/2.0/dbfs/create":
		f.next++
		f.handles[f.next] = req["path"].(string)
		f.files[req["path"].(string)] = nil
		reply(map[string]any{"handle": f.next})
	case "/api/2.0/dbfs/add-block":
		var data []byte
		_ = json.Unmarshal([]byte(`"`+req["data"].(string)+`"`), &data)
		p := f.handles[int64(req["handle"].(float64))]
		f.files[p] = append(f.files[p], data...)
		reply(map[string]any{})
	case "/api/2.0/dbfs/close":
		reply(map[string]any{})
	case "/api/2.0/dbfs/get-status":
		data, ok := f.files[q.Get("path")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			reply(map[string]any{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "nope"})
			return
		}
		reply(map[string]any{"path": q.Get("path"), "file_size": len(data)})
	case "/api/2.0/dbfs/read":
		data := f.files[q.Get("path")]
		var off, n int
		_ = json.Unmarshal([]byte(q.Get("offset")), &off)
		_ = json.Unmarshal([]byte(q.Get("length")), &n)
		chunk := data[min(off, len(data)):min(off+n, len(data))]
		reply(map[string]any{"bytes_read": len(chunk), "data": chunk})
	default:
		http.NotFound(w, r)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	fake := &fakeDBFS{files: map[string][]byte{}, handles: map[int64]string{}}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	ctx := context.Background()
	c, err := sdkclients.New(ctx, &config.Config{Host: srv.URL, Token: "t", Loaders: []config.Loader{config.ConfigAttributes}})
	require.NoError(t, err)

	content := bytes.Repeat([]byte("abc"), maxBlockSize)
	require.NoError(t, WriteFile(ctx, c.Files, "/tmp/x", content))
	assert.Equal(t, content, fake.files["/tmp/x"])

	got, err := ReadFile(ctx, c.Files, "/tmp/x")
	require.NoError(t, err)
	assert.Equal(t, content, got)

	_, err = Open(ctx, c.Files, "/tmp/missing", FileModeRead)
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "dbfs open: "))
}
