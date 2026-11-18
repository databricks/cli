package fs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	sdkapierr "github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func apiError(status int, header http.Header, body string) error {
	return apierr.FromHTTPError(status, header, []byte(body))
}

func TestClassifyError(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()

	jsonBody := `{"error_code": "INJECTED", "message": "boom"}`
	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want errorClass
	}{
		{"500 with empty body", t.Context(), apiError(500, nil, ""), classRetryable},
		{"500 with non-JSON body", t.Context(), apiError(500, nil, "<html>oops</html>"), classRetryable},
		{"503 wrapped", t.Context(), fmt.Errorf("write: %w", apiError(503, nil, jsonBody)), classRetryable},
		{"429", t.Context(), apiError(429, nil, jsonBody), classRetryable},
		{"408", t.Context(), apiError(408, nil, jsonBody), classRetryable},
		{"DBFS 500", t.Context(), &sdkapierr.APIError{StatusCode: 500}, classRetryable},
		{"400", t.Context(), apiError(400, nil, jsonBody), classFile},
		{"404", t.Context(), apiError(404, nil, jsonBody), classFile},
		{"DBFS 404", t.Context(), &sdkapierr.APIError{StatusCode: 404}, classFile},
		{"401", t.Context(), apiError(401, nil, jsonBody), classFatal},
		{"403", t.Context(), apiError(403, nil, jsonBody), classFatal},
		{"DBFS 403", t.Context(), &sdkapierr.APIError{StatusCode: 403}, classFatal},
		{"connection reset", t.Context(), fmt.Errorf("put: %w", syscall.ECONNRESET), classRetryable},
		{"unexpected EOF", t.Context(), io.ErrUnexpectedEOF, classRetryable},
		{"HTTP/2 stream refused", t.Context(), &url.Error{Op: "Put", URL: "u", Err: errors.New("stream error: REFUSED_STREAM")}, classRetryable},
		{"request timeout", t.Context(), fmt.Errorf("put: %w", context.DeadlineExceeded), classRetryable},
		{"copy cancelled", cancelled, apiError(503, nil, jsonBody), classFatal},
		{"local read error", t.Context(), errors.New("read src/a: input/output error"), classFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, classifyError(tt.ctx, tt.err))
		})
	}
}

func TestDescribeError(t *testing.T) {
	assert.Equal(t, "HTTP 500 Internal Server Error", describeError(apiError(500, nil, "")))
	assert.Equal(t, "HTTP 502 Bad Gateway", describeError(apiError(502, nil, "<html>bad gateway</html>")))
	assert.Equal(t, "databricks-api error: boom (HTTP 503 Service Unavailable)", describeError(apiError(503, nil, `{"message": "boom"}`)))
	assert.Equal(t, "HTTP 500 Internal Server Error", describeError(&sdkapierr.APIError{StatusCode: 500}))
	assert.Equal(t, "read failed", describeError(errors.New("read failed")))
}

// names returns n file names: f00.txt, f01.txt, ...
func names(n int) []string {
	var out []string
	for i := range n {
		out = append(out, fmt.Sprintf("f%02d.txt", i))
	}
	return out
}

// testSource returns a source filer with the given files under "src". Each
// file holds 3 bytes.
func testSource(files ...string) *filer.FakeFiler {
	entries := map[string]fakefs.FileInfo{"src": {FakeDir: true}}
	for _, f := range files {
		p := path.Join("src", f)
		entries[p] = fakefs.FileInfo{FakeSize: 3}
		for d := path.Dir(p); d != "src"; d = path.Dir(d) {
			entries[d] = fakefs.FileInfo{FakeDir: true}
		}
	}
	return filer.NewFakeFiler(entries)
}

// errCancel makes a scripted write cancel the copy's context.
var errCancel = errors.New("cancel")

// scriptedTarget is a target filer whose writes return what write returns for
// the path and the attempt on it, counting from 1. A Stat fails the test unless
// the caller sets one.
type scriptedTarget struct {
	mockFiler

	mu       sync.Mutex
	attempts map[string]int
	order    []string
}

func newScriptedTarget(t *testing.T, cancel context.CancelFunc, write func(ctx context.Context, p string, attempt int) error) *scriptedTarget {
	st := &scriptedTarget{attempts: map[string]int{}}
	st.stat = func(ctx context.Context, p string) (fs.FileInfo, error) {
		t.Errorf("unexpected Stat of %s", p)
		return nil, fs.ErrNotExist
	}
	st.write = func(ctx context.Context, p string, r io.Reader, mode ...filer.WriteMode) error {
		st.mu.Lock()
		st.attempts[p]++
		attempt := st.attempts[p]
		st.order = append(st.order, p)
		st.mu.Unlock()
		if write == nil {
			return nil
		}
		err := write(ctx, p, attempt)
		if errors.Is(err, errCancel) {
			cancel()
			return context.Canceled
		}
		return err
	}
	return st
}

type copyResult struct {
	r      *recursiveCopy
	err    error
	stdout string
	stderr string
	log    string
}

// runCopy copies "src" to "dst", capturing stdout, stderr and the debug log.
// opts adjust the copy before it runs.
func runCopy(t *testing.T, ctx context.Context, c *copy, stdout io.Writer, opts ...func(*recursiveCopy)) copyResult {
	var out, stderr, logs bytes.Buffer
	if stdout == nil {
		stdout = &out
	}
	format := flags.OutputText
	if c.outputJSON {
		format = flags.OutputJSON
	}
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, format, strings.NewReader(""), stdout, &stderr, "", ""))
	ctx = log.NewContext(ctx, slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	// Ignore any override in the environment: tests assume the defaults.
	ctx = env.Set(ctx, stallTimeoutEnvVar, "")
	ctx = env.Set(ctx, probeBackoffMaxEnvVar, "")
	r := newRecursiveCopy(ctx, c)
	for _, opt := range opts {
		opt(r)
	}

	done := make(chan error, 1)
	go func() { done <- r.run(ctx, "src", "dst") }()
	select {
	case err := <-done:
		return copyResult{r: r, err: err, stdout: out.String(), stderr: stderr.String(), log: logs.String()}
	case <-time.After(10 * time.Second):
		t.Fatal("recursive copy did not finish")
		return copyResult{}
	}
}

// brokenPipe is stdout after the reader went away.
type brokenPipe struct{}

var errBrokenPipe = errors.New("write |1: broken pipe")

func (brokenPipe) Write([]byte) (int, error) { return 0, errBrokenPipe }

func TestRecursiveCopy(t *testing.T) {
	notFound := apiError(404, nil, `{"error_code": "NOT_FOUND", "message": "gone"}`)
	sized := func(size int64) func(ctx context.Context, p string) (fs.FileInfo, error) {
		return func(ctx context.Context, p string) (fs.FileInfo, error) {
			return fakefs.FileInfo{FakeName: path.Base(p), FakeSize: size}, nil
		}
	}
	// failsOnceThen makes f01 fail with a 500 on its first attempt, which
	// defers it, and return err on its retry.
	failsOnceThen := func(err error) func(ctx context.Context, p string, attempt int) error {
		return func(ctx context.Context, p string, attempt int) error {
			switch {
			case p != "dst/f01.txt":
				return nil
			case attempt == 1:
				return apiError(500, nil, "")
			}
			return err
		}
	}
	only := func(file string, err error) func(ctx context.Context, p string, attempt int) error {
		return func(ctx context.Context, p string, attempt int) error {
			if p == file {
				return err
			}
			return nil
		}
	}

	var failedTwelve strings.Builder
	failedTwelve.WriteString("Copy incomplete: 0 copied, 0 skipped (already existed), 12 failed, 0 not attempted.\nFailed:\n")
	for _, f := range names(10) {
		fmt.Fprintf(&failedTwelve, "  src/%s -> dst/%s: databricks-api error: gone (HTTP 404 Not Found)\n", f, f)
	}
	failedTwelve.WriteString("  ... and 2 more\n")

	tests := []struct {
		name        string
		files       []string
		concurrency int
		json        bool
		scheme      string
		brokenPipe  bool
		write       func(ctx context.Context, p string, attempt int) error
		targetStat  func(ctx context.Context, p string) (fs.FileInfo, error)
		mkdir       func(ctx context.Context, p string) error

		wantErr      string
		wantStderr   string
		wantAttempts map[string]int
		wantLog      string
	}{
		{
			name:       "all copied",
			files:      names(5),
			wantStderr: "Copied 5 files.\n",
		},
		{
			name:       "one file",
			files:      names(1),
			wantStderr: "Copied 1 file.\n",
		},
		{
			name:       "JSON output leaves out the line for a complete copy",
			files:      names(3),
			json:       true,
			wantStderr: "",
		},
		{
			// A file that existed before the copy is skipped without a Stat.
			name:       "existing file is skipped",
			files:      names(3),
			write:      only("dst/f00.txt", fs.ErrExist),
			wantStderr: "Copied 2 files, skipped 1 that already existed.\n",
		},
		{
			name:         "a 500 is retried after the walk",
			files:        names(4),
			write:        failsOnceThen(nil),
			wantStderr:   "Copied 4 files.\n",
			wantAttempts: map[string]int{"dst/f01.txt": 2},
			wantLog:      "fs cp: deferring src/f01.txt: HTTP 500 Internal Server Error",
		},
		{
			name:         "a file still failing on its retry fails alone",
			files:        names(4),
			write:        failsOnceThen(apiError(503, nil, "")),
			wantErr:      "1 file was not copied",
			wantStderr:   "Copy incomplete: 3 copied, 0 skipped (already existed), 1 failed, 0 not attempted.\nFailed:\n  src/f01.txt -> dst/f01.txt: HTTP 503 Service Unavailable\n",
			wantAttempts: map[string]int{"dst/f01.txt": 2},
			wantLog:      "fs cp: src/f01.txt failed on its retry: HTTP 503 Service Unavailable",
		},
		{
			name:         "a 404 fails only that file, without a retry",
			files:        names(4),
			write:        only("dst/f01.txt", notFound),
			wantErr:      "1 file was not copied",
			wantStderr:   "Copy incomplete: 3 copied, 0 skipped (already existed), 1 failed, 0 not attempted.\nFailed:\n  src/f01.txt -> dst/f01.txt: databricks-api error: gone (HTTP 404 Not Found)\n",
			wantAttempts: map[string]int{"dst/f01.txt": 1},
		},
		{
			// The walk goes on to count f02 to f04.
			name:        "a 401 stops the copy",
			files:       names(5),
			concurrency: 1,
			write:       only("dst/f01.txt", apiError(401, nil, `{"message": "token expired"}`)),
			wantErr:     "databricks-api error: token expired",
			wantStderr:  "Copy incomplete: 1 copied, 0 skipped (already existed), 1 failed, 3 not attempted.\nStopped early because of the error shown below.\nFailed:\n  src/f01.txt -> dst/f01.txt: databricks-api error: token expired (HTTP 401 Unauthorized)\n",
		},
		{
			name:        "lists the first 10 failures",
			files:       names(12),
			concurrency: 1,
			write:       func(context.Context, string, int) error { return notFound },
			wantErr:     "12 files were not copied",
			wantStderr:  failedTwelve.String(),
		},
		{
			name:       "shows schemes",
			files:      names(1),
			scheme:     "dbfs",
			write:      only("dst/f00.txt", notFound),
			wantErr:    "1 file was not copied",
			wantStderr: "Copy incomplete: 0 copied, 0 skipped (already existed), 1 failed, 0 not attempted.\nFailed:\n  src/f00.txt -> dbfs:/dst/f00.txt: databricks-api error: gone (HTTP 404 Not Found)\n",
		},
		{
			// The first attempt landed even though it returned an error.
			name:       "a retry finding a target of the same size skips it",
			files:      names(2),
			write:      failsOnceThen(fs.ErrExist),
			targetStat: sized(3),
			wantStderr: "Copied 1 file, skipped 1 that already existed.\n",
		},
		{
			name:       "a retry finding a target of another size fails it",
			files:      names(2),
			write:      failsOnceThen(fs.ErrExist),
			targetStat: sized(1),
			wantErr:    "1 file was not copied",
			wantStderr: "Copy incomplete: 1 copied, 0 skipped (already existed), 1 failed, 0 not attempted.\nFailed:\n  src/f01.txt -> dst/f01.txt: target exists with a different size; an earlier attempt may have left a partial file. Rerun with --overwrite to replace it\n",
		},
		{
			name:       "a retry failing to Stat the target fails it",
			files:      names(2),
			write:      failsOnceThen(fs.ErrExist),
			targetStat: func(context.Context, string) (fs.FileInfo, error) { return nil, apiError(503, nil, "") },
			wantErr:    "1 file was not copied",
			wantStderr: "Copy incomplete: 1 copied, 0 skipped (already existed), 1 failed, 0 not attempted.\nFailed:\n  src/f01.txt -> dst/f01.txt: HTTP 503 Service Unavailable\n",
		},
		{
			name:  "a 403 from that Stat stops the copy",
			files: names(2),
			write: failsOnceThen(fs.ErrExist),
			targetStat: func(context.Context, string) (fs.FileInfo, error) {
				return nil, apiError(403, nil, `{"message": "denied"}`)
			},
			wantErr:    "databricks-api error: denied",
			wantStderr: "Copy incomplete: 1 copied, 0 skipped (already existed), 1 failed, 0 not attempted.\nStopped early because of the error shown below.\nFailed:\n  src/f01.txt -> dst/f01.txt: databricks-api error: denied (HTTP 403 Forbidden)\n",
		},
		{
			// f01 is cancelled mid-copy; f02 and f03 are counted but not
			// started.
			name:        "cancelling the copy stops it",
			files:       names(4),
			concurrency: 1,
			write:       only("dst/f01.txt", errCancel),
			wantErr:     "copy cancelled: context canceled",
			wantStderr:  "Copy incomplete: 1 copied, 0 skipped (already existed), 1 failed, 2 not attempted.\nStopped: the copy was cancelled.\nFailed:\n  src/f01.txt -> dst/f01.txt: cancelled: the copy stopped before this file finished\n",
		},
		{
			name:       "a deferred file cancelled on its retry keeps its error",
			files:      names(3),
			write:      failsOnceThen(errCancel),
			wantErr:    "copy cancelled: context canceled",
			wantStderr: "Copy incomplete: 2 copied, 0 skipped (already existed), 1 failed, 0 not attempted.\nStopped: the copy was cancelled.\nFailed:\n  src/f01.txt -> dst/f01.txt: HTTP 500 Internal Server Error\n",
		},
		{
			name:        "failing to print stops the copy",
			files:       names(4),
			concurrency: 1,
			brokenPipe:  true,
			wantErr:     errBrokenPipe.Error(),
			wantStderr:  "Copy incomplete: 0 copied, 0 skipped (already existed), 1 failed, 3 not attempted.\nStopped early because of the error shown below.\nFailed:\n  src/f00.txt -> dst/f00.txt: write |1: broken pipe\n",
		},
		{
			// The copy of a.txt waits until it is cancelled. The walk ends at
			// the error, so files it never reached are not counted.
			name:  "a failed Mkdir stops the copy and cancels in-flight files",
			files: []string{"a.txt", "sub/b.txt"},
			write: func(ctx context.Context, p string, attempt int) error {
				<-ctx.Done()
				return ctx.Err()
			},
			mkdir: func(ctx context.Context, p string) error {
				if p == "dst/sub" {
					return apiError(400, nil, `{"message": "bad path"}`)
				}
				return nil
			},
			wantErr:    "databricks-api error: bad path",
			wantStderr: "Copy incomplete: 0 copied, 0 skipped (already existed), 1 failed, at least 0 not attempted.\nStopped early because of the error shown below.\nFailed:\n  src/a.txt -> dst/a.txt: cancelled: the copy stopped before this file finished\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			target := newScriptedTarget(t, cancel, tt.write)
			if tt.targetStat != nil {
				target.stat = tt.targetStat
			}
			target.mkdir = tt.mkdir
			c := &copy{
				recursive:    true,
				concurrency:  orDefault(tt.concurrency, 2),
				outputJSON:   tt.json,
				targetScheme: tt.scheme,
				sourceFiler:  testSource(tt.files...),
				targetFiler:  target,
			}
			var stdout io.Writer
			if tt.brokenPipe {
				stdout = brokenPipe{}
			}
			res := runCopy(t, ctx, c, stdout)

			if tt.wantErr == "" {
				assert.NoError(t, res.err)
			} else {
				assert.EqualError(t, res.err, tt.wantErr)
			}
			assert.Equal(t, tt.wantStderr, res.stderr)
			if !tt.json && !tt.brokenPipe {
				assert.Equal(t, res.r.copied+res.r.skipped, strings.Count(res.stdout, "\n"), "one stdout line per copied or skipped file")
			}
			for p, n := range tt.wantAttempts {
				assert.Equal(t, n, target.attempts[p], "attempts on %s", p)
			}
			if tt.wantAttempts["dst/f01.txt"] == 2 {
				assert.Equal(t, "dst/f01.txt", target.order[len(target.order)-1], "the retry comes after the walk")
			}
			assert.Contains(t, res.log, tt.wantLog)
		})
	}
}

// orDefault returns v, or def if v is zero.
func orDefault(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

// flakySource is a source filer whose ReadDir of dir returns what readDir
// returns for the attempt, counting from 1.
type flakySource struct {
	*filer.FakeFiler
	dir      string
	readDir  func(attempt int) error
	attempts int
}

func (f *flakySource) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	if name == f.dir {
		f.attempts++
		if err := f.readDir(f.attempts); err != nil {
			return nil, err
		}
	}
	return f.FakeFiler.ReadDir(ctx, name)
}

// fakeClock is a clock that only moves when the test advances it or the code
// under test sleeps.
type fakeClock struct {
	mu     sync.Mutex
	t      time.Time
	sleeps []time.Duration
}

func (c *fakeClock) install(r *recursiveCopy) {
	r.now = c.Now
	r.sleep = c.Sleep
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
	c.sleeps = append(c.sleeps, d)
	return nil
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestRecursiveCopyStall(t *testing.T) {
	unavailable := apiError(503, nil, "")
	stalledOne := func(notAttempted string) string {
		return "Copy incomplete: 0 copied, 0 skipped (already existed), 1 failed, " + notAttempted + " not attempted.\n" +
			"Stopped: files kept failing and none was copied for 5 minutes, so no new files were started.\nFailed:\n" +
			"  src/f00.txt -> dst/f00.txt: HTTP 503 Service Unavailable\n"
	}

	tests := []struct {
		name        string
		files       []string
		failListing string
		listTime    time.Duration
		mkdirTime   time.Duration
		writeTime   time.Duration
		fail        func(p string, attempt int) bool
		wantErr     string
		wantStderr  string
	}{
		{
			// Each write fails after a minute. f00 fails and opens the
			// breaker, which probes it until files have kept failing for 5
			// minutes. Then no new files start, and f00 keeps its error.
			name:       "an outage stops the copy",
			files:      names(20),
			writeTime:  time.Minute,
			fail:       func(string, int) bool { return true },
			wantErr:    "copy stopped: files kept failing and none was copied for 5 minutes",
			wantStderr: stalledOne("19"),
		},
		{
			// While the stalled copy only counts the files left, a failed
			// listing ends the count but is not why the copy stopped.
			name:        "a listing error while counting keeps the stall as the reason",
			files:       append(names(20), "z/x.txt"),
			failListing: "src/z",
			writeTime:   time.Minute,
			fail:        func(string, int) bool { return true },
			wantErr:     "copy stopped: files kept failing and none was copied for 5 minutes",
			wantStderr:  stalledOne("at least 19"),
		},
		{
			// Writes take 2 minutes and every other one fails on its first
			// attempt. The copy takes 12 minutes, but the gap between
			// successes stays under 5.
			name:      "a success resets the timer",
			files:     names(4),
			writeTime: 2 * time.Minute,
			fail: func(p string, attempt int) bool {
				return attempt == 1 && (p == "dst/f01.txt" || p == "dst/f03.txt")
			},
			wantStderr: "Copied 4 files.\n",
		},
		{
			// Each Mkdir takes 6 minutes. a.txt and b.txt fail on their first
			// attempt (with one worker, b.txt starts only after a.txt failed),
			// so d1/c.txt starts 6 minutes after the first failure. Creating
			// d1 in between counts as progress.
			name:      "slow directories count as progress",
			files:     []string{"a.txt", "b.txt", "d1/c.txt"},
			mkdirTime: 6 * time.Minute,
			fail: func(p string, attempt int) bool {
				return attempt == 1 && (p == "dst/a.txt" || p == "dst/b.txt")
			},
			wantStderr: "Copied 3 files.\n",
		},
		{
			// Listing sub and each write take 6 minutes, with no failures.
			name:       "a slow copy with no failures never stalls",
			files:      []string{"sub/a.txt", "sub/b.txt"},
			listTime:   6 * time.Minute,
			writeTime:  6 * time.Minute,
			wantStderr: "Copied 2 files.\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			clock := &fakeClock{}
			var startedLate []string
			target := newScriptedTarget(t, cancel, func(ctx context.Context, p string, attempt int) error {
				// In the stalling rows the first write fails after writeTime,
				// and the timeout runs from that failure.
				if clock.Now().Sub(time.Time{}) >= tt.writeTime+stallTimeout {
					startedLate = append(startedLate, p)
				}
				clock.Advance(tt.writeTime)
				if tt.fail != nil && tt.fail(p, attempt) {
					return unavailable
				}
				return nil
			})
			target.mkdir = func(context.Context, string) error {
				clock.Advance(tt.mkdirTime)
				return nil
			}
			var source filer.Filer = testSource(tt.files...)
			switch {
			case tt.failListing != "":
				source = &flakySource{FakeFiler: testSource(tt.files...), dir: tt.failListing, readDir: func(int) error { return unavailable }}
			case tt.listTime > 0:
				source = &flakySource{FakeFiler: testSource(tt.files...), dir: "src/sub", readDir: func(int) error {
					clock.Advance(tt.listTime)
					return nil
				}}
			}
			c := &copy{recursive: true, concurrency: 1, sourceFiler: source, targetFiler: target}
			res := runCopy(t, ctx, c, nil, clock.install)

			if tt.wantErr == "" {
				assert.NoError(t, res.err)
				assert.NotContains(t, res.log, "starting no new files")
			} else {
				assert.EqualError(t, res.err, tt.wantErr)
				assert.Contains(t, res.log, "level=INFO msg=\"fs cp: files kept failing and none was copied for 5 minutes; starting no new files\"")
				assert.Empty(t, startedLate, "no file starts after the stall timeout")
			}
			assert.Equal(t, tt.wantStderr, res.stderr)
		})
	}
}

func TestPermanentErrorsDoNotStall(t *testing.T) {
	// f00.txt fails with a 404, then listing sub takes 6 minutes with nothing
	// completing. A permanent error is not an outage, so sub/a.txt still runs.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	// A real clock is never at the zero time, which failingSince uses to mean
	// "not failing".
	clock := &fakeClock{t: time.Time{}.Add(time.Hour)}
	target := newScriptedTarget(t, cancel, func(ctx context.Context, p string, attempt int) error {
		if p == "dst/f00.txt" {
			return apiError(404, nil, `{"message": "gone"}`)
		}
		return nil
	})
	var r *recursiveCopy
	source := &flakySource{FakeFiler: testSource("f00.txt", "sub/a.txt"), dir: "src/sub", readDir: func(int) error {
		// The 404 is recorded on another goroutine; wait for it, so it comes
		// before the slow listing.
		for {
			r.mu.Lock()
			n := len(r.failures)
			r.mu.Unlock()
			if n > 0 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		clock.Advance(6 * time.Minute)
		return nil
	}}
	c := &copy{recursive: true, concurrency: 1, sourceFiler: source, targetFiler: target}
	res := runCopy(t, ctx, c, nil, func(rc *recursiveCopy) {
		rc.now = clock.Now
		r = rc
	})

	assert.EqualError(t, res.err, "1 file was not copied")
	assert.Equal(t, 1, res.r.copied)
	assert.NotContains(t, res.log, "starting no new files")
}

func TestStallTimesTheFailingPeriod(t *testing.T) {
	// The last success was at 0, and a long healthy stretch with nothing
	// completing ends in a failure at 6 minutes. That alone must not stall the
	// copy: the timeout runs from the failure.
	clock := &fakeClock{}
	r := newRecursiveCopy(t.Context(), &copy{})
	r.now = clock.Now
	r.lastSuccess = clock.Now()
	assert.False(t, r.stopping(t.Context()), "no failure, no stall")

	clock.Advance(6 * time.Minute)
	r.failingSince = clock.Now()
	assert.False(t, r.stopping(t.Context()), "a failure just now")

	clock.Advance(5 * time.Minute)
	assert.True(t, r.stopping(t.Context()), "failing for the stall timeout")
}

func TestStallTimeoutFromEnv(t *testing.T) {
	for value, want := range map[string]time.Duration{
		"":     stallTimeout,
		"1500": 1500 * time.Millisecond,
		"0":    stallTimeout,
		"5m":   stallTimeout,
	} {
		ctx := env.Set(t.Context(), stallTimeoutEnvVar, value)
		assert.Equal(t, want, newRecursiveCopy(ctx, &copy{}).stallTimeout, "value %q", value)
	}
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "5 minutes", formatDuration(5*time.Minute))
	assert.Equal(t, "1 minute", formatDuration(time.Minute))
	assert.Equal(t, "90 seconds", formatDuration(90*time.Second))
	assert.Equal(t, "1.5s", formatDuration(1500*time.Millisecond))
}

func TestRecursiveCopyBreakerBackoff(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter string
		wantSleeps []time.Duration
	}{
		{
			name:       "exponential from 2s",
			wantSleeps: []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second},
		},
		{
			name:       "Retry-After above the backoff is honored",
			retryAfter: "5",
			wantSleeps: []time.Duration{5 * time.Second, 5 * time.Second, 8 * time.Second, 16 * time.Second},
		},
		{
			name:       "Retry-After is capped at 30s",
			retryAfter: "120",
			wantSleeps: []time.Duration{30 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			if tt.retryAfter != "" {
				header.Set("Retry-After", tt.retryAfter)
			}
			// The first 4 writes fail with 503. With one worker the breaker
			// opens after the first, and each probe retries the deferred f00
			// until one succeeds.
			var mu sync.Mutex
			failuresLeft := 4
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			target := newScriptedTarget(t, cancel, func(context.Context, string, int) error {
				mu.Lock()
				defer mu.Unlock()
				if failuresLeft > 0 {
					failuresLeft--
					return apiError(503, header, "")
				}
				return nil
			})
			c := &copy{recursive: true, concurrency: 1, sourceFiler: testSource(names(3)...), targetFiler: target}
			clock := &fakeClock{}
			res := runCopy(t, ctx, c, nil, clock.install)

			require.NoError(t, res.err)
			assert.Equal(t, "Copied 3 files.\n", res.stderr)
			assert.Equal(t, tt.wantSleeps, clock.sleeps)
			assert.Equal(t, []string{"dst/f00.txt", "dst/f00.txt", "dst/f00.txt", "dst/f00.txt", "dst/f00.txt", "dst/f01.txt", "dst/f02.txt"}, target.order)
			for _, line := range []string{
				`level=INFO msg="fs cp: breaker opened after 1 retryable failures in a row (last: HTTP 503 Service Unavailable); trying one file at a time"`,
				"fs cp: probe with src/f00.txt failed: HTTP 503 Service Unavailable",
				"fs cp: probe with src/f00.txt succeeded",
				`level=INFO msg="fs cp: breaker closed: src/f00.txt was copied; starting new files again"`,
			} {
				assert.Contains(t, res.log, line)
			}
		})
	}
}

func TestRecursiveCopyBreakerOnDeferredPass(t *testing.T) {
	// f00 and f01 fail on their first attempt, which opens the breaker
	// (--concurrency 2). On the deferred pass, the first retry is the probe:
	// it runs after a backoff, succeeds and closes the breaker, and the second
	// retry starts normally.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	target := newScriptedTarget(t, cancel, func(ctx context.Context, p string, attempt int) error {
		if attempt == 1 {
			return apiError(503, nil, "")
		}
		return nil
	})
	c := &copy{recursive: true, concurrency: 2, sourceFiler: testSource(names(2)...), targetFiler: target}
	clock := &fakeClock{}
	res := runCopy(t, ctx, c, nil, clock.install)

	require.NoError(t, res.err)
	assert.Equal(t, "Copied 2 files.\n", res.stderr)
	assert.Equal(t, []time.Duration{2 * time.Second}, clock.sleeps)
	assert.Equal(t, map[string]int{"dst/f00.txt": 2, "dst/f01.txt": 2}, target.attempts)
	assert.Contains(t, res.log, "fs cp: breaker open, probing with src/f0")
	assert.Contains(t, res.log, "fs cp: breaker closed")
}

func TestBreakerOpensAndCloses(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	hint := http.Header{"Retry-After": []string{"9"}}
	steps := []struct {
		file     string
		err      error
		wantOpen bool
	}{
		// With --concurrency 2, a success between failures keeps it closed.
		{"f00.txt", apiError(503, nil, ""), false},
		{"f01.txt", nil, false},
		{"f02.txt", apiError(500, nil, ""), false},
		// Two in a row open it. A later failure without a hint keeps the
		// Retry-After of the one before.
		{"f03.txt", apiError(503, hint, ""), true},
		{"f04.txt", apiError(500, nil, ""), true},
		{"f05.txt", nil, false},
	}
	errs := map[string]error{}
	var files []string
	for _, step := range steps {
		errs["dst/"+step.file] = step.err
		files = append(files, step.file)
	}
	c := &copy{concurrency: 2, sourceFiler: testSource(files...)}
	c.targetFiler = newScriptedTarget(t, nil, func(ctx context.Context, p string, attempt int) error { return errs[p] })
	r := newRecursiveCopy(ctx, c)
	r.cancel = func() {}

	for _, step := range steps {
		_ = r.copyOne(ctx, copyJob{sourcePath: "src/" + step.file, targetPath: "dst/" + step.file})
		assert.Equal(t, step.wantOpen, r.breakerOpen, "after %s", step.file)
		if step.file == "f04.txt" {
			delay, open := r.breakerDelay()
			assert.True(t, open)
			assert.Equal(t, 9*time.Second, delay)
		}
	}
	assert.Len(t, r.deferred, 4)
}

func TestRetryAfter(t *testing.T) {
	header := http.Header{"Retry-After": []string{"7"}}
	assert.Equal(t, 7*time.Second, retryAfter(apiError(503, header, "")))
	assert.Equal(t, time.Duration(0), retryAfter(apiError(503, nil, "")))
	dbfs := &sdkapierr.APIError{StatusCode: 503, ResponseWrapper: &common.ResponseWrapper{Response: &http.Response{Header: header}}}
	assert.Equal(t, 7*time.Second, retryAfter(dbfs))
}

func TestProbeBackoffFromEnv(t *testing.T) {
	r := newRecursiveCopy(t.Context(), &copy{})
	assert.Equal(t, probeBackoffMin, r.probeBackoffMin)
	assert.Equal(t, probeBackoffMax, r.probeBackoffMax)

	r = newRecursiveCopy(env.Set(t.Context(), probeBackoffMaxEnvVar, "100"), &copy{})
	assert.Equal(t, 100*time.Millisecond, r.probeBackoffMax)
	assert.Equal(t, 100*time.Millisecond, r.probeBackoffMin, "a lower maximum lowers the minimum")
}

func TestRecursiveCopyWalkRetry(t *testing.T) {
	unavailable := apiError(503, nil, "")
	sec := func(n ...int) []time.Duration {
		var out []time.Duration
		for _, v := range n {
			out = append(out, time.Duration(v)*time.Second)
		}
		return out
	}
	tests := []struct {
		name       string
		files      []string
		listDir    string
		readDir    func(attempt int) error
		mkdir      func(p string, attempt int) error
		write      func(clock *fakeClock, p string, attempt int) error
		wantErr    error
		wantStderr string
		wantSleeps []time.Duration
		wantLog    string
	}{
		{
			name: "a Mkdir 500 is retried in place",
			mkdir: func(p string, attempt int) error {
				if p == "dst/sub" && attempt <= 2 {
					return apiError(500, nil, "")
				}
				return nil
			},
			wantStderr: "Copied 2 files.\n",
			wantSleeps: sec(2, 4),
			wantLog:    "fs cp: retrying create directory dst/sub in 2s after: HTTP 500 Internal Server Error",
		},
		{
			// The listing recovers after 4m30s of retries, then b.txt takes a
			// minute and fails once. Its retry still runs, because the
			// recovered listing counts as progress.
			name:  "a listing recovering after a long retry leaves the files their stall timeout",
			files: []string{"sub/b.txt"},
			readDir: func(attempt int) error {
				if attempt <= 12 {
					return unavailable
				}
				return nil
			},
			write: func(clock *fakeClock, p string, attempt int) error {
				if p != "dst/sub/b.txt" {
					return nil
				}
				clock.Advance(time.Minute)
				if attempt == 1 {
					return unavailable
				}
				return nil
			},
			wantStderr: "Copied 1 file.\n",
			wantSleeps: sec(2, 4, 8, 16, 30, 30, 30, 30, 30, 30, 30, 30, 2),
			wantLog:    "fs cp: retrying list src/sub in 2s after: HTTP 503 Service Unavailable",
		},
		{
			// The backoff is capped at 30s, and the retries add up to the
			// stall timeout.
			name:       "a listing failing for the stall timeout stops the copy",
			listDir:    "src",
			readDir:    func(int) error { return unavailable },
			wantErr:    unavailable,
			wantStderr: "Copy incomplete: 0 copied, 0 skipped (already existed), 0 failed, at least 0 not attempted.\nStopped early because of the error shown below.\n",
			wantSleeps: sec(2, 4, 8, 16, 30, 30, 30, 30, 30, 30, 30, 30, 30),
			wantLog:    "fs cp: retrying list src in 2s after: HTTP 503 Service Unavailable",
		},
		{
			// a.txt fails after 6 minutes, which opens the breaker, and so does
			// its probe 2s later, so b.txt finds the copy stalled. The walk then
			// only counts files, and does not retry the failed listing of sub:
			// the only sleep is the breaker's.
			name:    "a stalled copy does not retry listings",
			files:   []string{"a.txt", "b.txt", "sub/c.txt"},
			readDir: func(int) error { return unavailable },
			write: func(clock *fakeClock, p string, attempt int) error {
				clock.Advance(6 * time.Minute)
				return unavailable
			},
			wantErr:    errors.New("copy stopped: files kept failing and none was copied for 5 minutes"),
			wantStderr: "Copy incomplete: 0 copied, 0 skipped (already existed), 1 failed, at least 1 not attempted.\nStopped: files kept failing and none was copied for 5 minutes, so no new files were started.\nFailed:\n  src/a.txt -> dst/a.txt: HTTP 503 Service Unavailable\n",
			wantSleeps: sec(2),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			clock := &fakeClock{}
			files := tt.files
			if files == nil {
				files = []string{"a.txt", "sub/b.txt"}
			}
			source := &flakySource{FakeFiler: testSource(files...), dir: "src/sub", readDir: func(int) error { return nil }}
			if tt.listDir != "" {
				source.dir = tt.listDir
			}
			if tt.readDir != nil {
				source.readDir = tt.readDir
			}
			target := newScriptedTarget(t, cancel, func(ctx context.Context, p string, attempt int) error {
				if tt.write == nil {
					return nil
				}
				return tt.write(clock, p, attempt)
			})
			mkdirs := map[string]int{}
			target.mkdir = func(ctx context.Context, p string) error {
				mkdirs[p]++
				if tt.mkdir == nil {
					return nil
				}
				return tt.mkdir(p, mkdirs[p])
			}
			c := &copy{recursive: true, concurrency: 1, sourceFiler: source, targetFiler: target}
			res := runCopy(t, ctx, c, nil, clock.install)

			switch {
			case tt.wantErr == nil:
				assert.NoError(t, res.err)
			case errors.Is(tt.wantErr, unavailable):
				assert.ErrorIs(t, res.err, unavailable)
			default:
				assert.EqualError(t, res.err, tt.wantErr.Error())
			}
			assert.Equal(t, tt.wantStderr, res.stderr)
			assert.Equal(t, tt.wantSleeps, clock.sleeps)
			assert.Contains(t, res.log, tt.wantLog)
		})
	}
}
