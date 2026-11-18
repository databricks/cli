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
	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	sdkapierr "github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/stretchr/testify/assert"
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
func runCopy(t *testing.T, ctx context.Context, c *copy, stdout io.Writer) copyResult {
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
	r := &recursiveCopy{c: c}

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
