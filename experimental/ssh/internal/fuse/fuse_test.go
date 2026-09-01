package fuse

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitFor and tick bound the wait for the revoke that KeepRegistered does on its
// own goroutine.
const (
	waitFor = 5 * time.Second
	tick    = 10 * time.Millisecond
)

func TestParseStatStartTime(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    uint64
		wantErr string
	}{
		{
			// A real line, padded past the starttime field.
			name:    "ordinary comm",
			content: "1234 (databricks) S 1 1234 1234 0 -1 4194560 5407 0 0 0 33 12 0 0 20 0 14 0 8613244 " + padding,
			want:    8613244,
		},
		{
			// The kernel does not escape comm, so a process named "weird ) name)"
			// makes every field index counted from the left wrong.
			name:    "comm containing spaces and parentheses",
			content: "1234 (weird ) name)) S 1 1234 1234 0 -1 4194560 5407 0 0 0 33 12 0 0 20 0 14 0 999 " + padding,
			want:    999,
		},
		{
			name:    "no comm field",
			content: "1234 databricks S 1",
			wantErr: "no comm field",
		},
		{
			name:    "truncated after comm",
			content: "1234 (databricks) S 1 1234",
			wantErr: "only 3 fields after comm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStatStartTime(tt.content)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// padding stands in for the fields a real /proc/<pid>/stat carries after
// starttime, none of which are read.
const padding = "1234 5678 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 17 1 0 0 0 0 0"

func TestParsePidNamespace(t *testing.T) {
	tests := []struct {
		link    string
		want    uint32
		wantErr string
	}{
		{link: "pid:[4026531836]", want: 4026531836},
		{link: "pid:[4026531836]\n", want: 4026531836},
		{link: "pid:[]", wantErr: "failed to parse the pid namespace id"},
		{link: "mnt:[4026531836]", wantErr: "unexpected pid namespace link"},
		{link: "pid:[4026531836", wantErr: "unexpected pid namespace link"},
		// The daemons take a uint32, so an id that does not fit has to fail
		// rather than silently truncate.
		{link: "pid:[4294967296]", wantErr: "failed to parse the pid namespace id"},
	}

	for _, tt := range tests {
		t.Run(tt.link, func(t *testing.T) {
			got, err := parsePidNamespace(tt.link)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSelfDescribesTheRunningProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/proc is Linux-only; the tunnel server only ever runs on Linux")
	}

	self, err := Self()
	require.NoError(t, err)

	assert.Equal(t, os.Getpid(), self.PID)
	assert.NotZero(t, self.PidNamespaceID, "the daemons reject a zero namespace id when Files CSI is on")
	assert.NotZero(t, self.StartTime, "the daemons reject a zero start time")
}

// daemon stands in for one FUSE daemon and records what it was asked to do.
type daemon struct {
	server *httptest.Server
	port   int

	mu       sync.Mutex
	requests []daemonRequest
	status   int
}

type daemonRequest struct {
	method string
	path   string
	body   request
}

func newDaemon(t *testing.T) *daemon {
	t.Helper()
	d := &daemon{status: http.StatusOK}
	d.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// assert, not require: this runs on the server's goroutine, where
		// FailNow would be invalid.
		raw, err := io.ReadAll(r.Body)
		if !assert.NoError(t, err) {
			return
		}
		var body request
		if !assert.NoError(t, json.Unmarshal(raw, &body)) {
			return
		}

		d.mu.Lock()
		d.requests = append(d.requests, daemonRequest{method: r.Method, path: r.URL.Path, body: body})
		status := d.status
		d.mu.Unlock()

		if status != http.StatusOK {
			http.Error(w, "the daemon said no", status)
		}
	}))
	t.Cleanup(d.server.Close)

	parsed, err := url.Parse(d.server.URL)
	require.NoError(t, err)
	d.port, err = strconv.Atoi(parsed.Port())
	require.NoError(t, err)
	return d
}

func (d *daemon) failWith(status int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status = status
}

func (d *daemon) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.requests)
}

func (d *daemon) at(i int) daemonRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.requests[i]
}

// newTestClient wires a client to a pair of stand-in daemons.
func newTestClient(t *testing.T) (*Client, *daemon, *daemon) {
	t.Helper()
	wsfs, ucFuse := newDaemon(t), newDaemon(t)
	return &Client{
		host:       "127.0.0.1",
		wsfsPort:   wsfs.port,
		ucFusePort: ucFuse.port,
		http:       wsfs.server.Client(),
	}, wsfs, ucFuse
}

func TestRegisterSendsBothDaemonsThePidAndCredentials(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	r := Registration{PID: 4242, PidNamespaceID: 4026531836, StartTime: 8613244}

	require.NoError(t, client.Register(t.Context(), r, "dapi-a-token", "1234567890"))

	require.Equal(t, 1, wsfs.count())
	require.Equal(t, 1, ucFuse.count())

	want := request{
		APIToken:       "dapi-a-token",
		ProcStartTime:  8613244,
		CommandOrigin:  "RemoteSshServer",
		PidNamespaceID: 4026531836,
		AdditionalTags: map[string]string{"userId": "1234567890"},
	}
	// The pid goes in the path, not the body, and the two daemons expose the
	// registration under different prefixes.
	assert.Equal(t, daemonRequest{method: http.MethodPut, path: "/api/1/pid/4242", body: want}, wsfs.at(0))
	assert.Equal(t, daemonRequest{method: http.MethodPut, path: "/dbfs-fuse-api/1/pid/4242", body: want}, ucFuse.at(0))
}

func TestRegisterWithoutAUserIdOmitsTheTags(t *testing.T) {
	client, wsfs, _ := newTestClient(t)
	r := Registration{PID: 1, PidNamespaceID: 2, StartTime: 3}

	require.NoError(t, client.Register(t.Context(), r, "token", ""))
	assert.Nil(t, wsfs.at(0).body.AdditionalTags)
}

func TestRegisterRejectsWhatTheDaemonsWouldMisread(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	r := Registration{PID: 1, PidNamespaceID: 2, StartTime: 3}

	// An empty token is how a revoke is spelled, so registering one would clear
	// the very access it was meant to grant.
	err := client.Register(t.Context(), r, "", "")
	assert.ErrorContains(t, err, "empty token")

	// The daemons answer 400 to a non-empty token with a zero start time; catching
	// it here makes the error name the cause.
	err = client.Register(t.Context(), Registration{PID: 1, PidNamespaceID: 2}, "token", "")
	assert.ErrorContains(t, err, "zero start time")

	assert.Zero(t, wsfs.count(), "nothing should have been sent")
	assert.Zero(t, ucFuse.count(), "nothing should have been sent")
}

func TestRevokeClearsTheRegistration(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	r := Registration{PID: 4242, PidNamespaceID: 4026531836, StartTime: 8613244}

	require.NoError(t, client.Revoke(t.Context(), r))

	// The daemons recognise a revoke by the empty token alone, and the start time
	// goes back to zero with it.
	want := request{CommandOrigin: "RemoteSshServer", PidNamespaceID: 4026531836}
	assert.Equal(t, daemonRequest{method: http.MethodPut, path: "/api/1/pid/4242", body: want}, wsfs.at(0))
	assert.Equal(t, daemonRequest{method: http.MethodPut, path: "/dbfs-fuse-api/1/pid/4242", body: want}, ucFuse.at(0))
}

func TestOneDaemonFailingStillRegistersTheOther(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	wsfs.failWith(http.StatusInternalServerError)

	err := client.Register(t.Context(), Registration{PID: 1, PidNamespaceID: 2, StartTime: 3}, "token", "")

	// /Volumes and /dbfs are still worth having when /Workspace cannot be
	// registered, so both are asked and the errors are reported together.
	assert.ErrorContains(t, err, "wsfs:")
	assert.ErrorContains(t, err, "500")
	assert.Equal(t, 1, ucFuse.count(), "uc-fuse should have been asked regardless")
}

func TestBothDaemonsFailingIsReportedAsBoth(t *testing.T) {
	client, wsfs, ucFuse := newTestClient(t)
	wsfs.failWith(http.StatusBadRequest)
	ucFuse.failWith(http.StatusBadRequest)

	err := client.Register(t.Context(), Registration{PID: 1, PidNamespaceID: 2, StartTime: 3}, "token", "")
	assert.ErrorContains(t, err, "wsfs:")
	assert.ErrorContains(t, err, "uc-fuse:")
}
