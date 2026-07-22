package files

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/tmp/auth"
	"github.com/databricks/sdk-go/core/ops"
	"github.com/databricks/cli/libs/tmp/options/client"
)

// stubCredentials is a no-op credential for tests: NewClient requires
// credentials, but these tests exercise the control plane through a fake server
// that ignores auth headers.
type stubCredentials struct{}

func (stubCredentials) Name() string                                       { return "stub" }
func (stubCredentials) AuthHeaders(context.Context) ([]auth.Header, error) { return nil, nil }

// buildUploadClient builds a v2 Client whose control plane is served by hc and
// addressed at host, with an optional workspace ID for the routing header.
func buildUploadClient(t *testing.T, host string, hc *http.Client, workspaceID string) *Client {
	t.Helper()
	opts := []client.Option{
		client.WithHost(host),
		client.WithHTTPClient(hc),
		client.WithCredentials(stubCredentials{}),
		client.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		// Keep tests hermetic: without this, an unset workspace ID would be
		// filled from the developer's local profile.
		client.WithoutProfileResolution(),
	}
	if workspaceID != "" {
		opts = append(opts, client.WithWorkspaceID(workspaceID))
	}
	c, err := NewClient(context.Background(), opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// fakeServer emulates both the Files API control plane and the cloud storage
// provider: the presigned URLs it mints point back at itself, so a single
// httptest server backs the whole upload. It reassembles the uploaded object so
// tests can assert byte-for-byte equality with the input.
type fakeServer struct {
	srv *httptest.Server

	mu          sync.Mutex
	mode        string // "multipart" or "resumable"
	parts       map[int][]byte
	completed   []completePart
	singleBody  []byte
	resumable   []byte
	overwriteQ  []string      // overwrite query values seen on initiate + single-shot
	partHeaders []http.Header // headers seen on cloud part PUTs
	partURLReqs []int         // Count of each create-upload-part-urls request

	// partHook injects faults. It returns (statusCode, etag, body); a zero
	// statusCode means "use the default 200 + store".
	partHook func(n, attempt int) (int, string, []byte)
	partTry  map[int]int

	// singleShotStatus, when non-zero, is returned by the single-shot PUT.
	singleShotStatus int
	singleShotBody   string

	// singleShotHook injects per-attempt faults on the single-shot PUT. It
	// returns the status for the given 1-based attempt; a zero status means "use
	// the default 200 + store". singleTry counts single-shot PUTs seen.
	singleShotHook func(attempt int) int
	singleTry      int

	// stuckResumable makes a resumable chunk PUT confirm no new bytes, modeling a
	// server/proxy wedged at a fixed offset.
	stuckResumable bool

	// resumableChunkHook injects per-chunk faults on the resumable data PUT. For a
	// non-zero returned status the chunk's bytes are still committed (so the
	// client's status query reveals forward progress), but the response is that
	// status -- modeling a spurious transient failure on a chunk that did land.
	// resumableTry counts data-chunk PUTs (not status queries).
	resumableChunkHook func(attempt int) int
	resumableTry       int
}

func newFakeServer(t *testing.T, mode string) *fakeServer {
	f := &fakeServer{
		mode:    mode,
		parts:   map[int][]byte{},
		partTry: map[int]int{},
	}
	f.srv = httptest.NewServer(f)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeServer) base() string { return f.srv.URL }

func (f *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/create-upload-part-urls"):
		f.handleCreatePartURLs(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/create-resumable-upload-url"):
		writeJSON(w, resumableURLResponse{ResumableUploadURL: &urlWithHeaders{URL: f.base() + "/cloud/resumable"}})
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/create-abort-upload-url"):
		writeJSON(w, abortURLResponse{AbortUploadURL: &urlWithHeaders{URL: f.base() + "/cloud/abort"}})
	case r.Method == http.MethodPut && strings.HasPrefix(path, "/cloud/part/"):
		f.handlePartPut(w, r)
	case r.Method == http.MethodPut && path == "/cloud/resumable":
		f.handleResumablePut(w, r)
	case r.Method == http.MethodDelete && (path == "/cloud/abort" || path == "/cloud/resumable"):
		w.WriteHeader(http.StatusOK)
	case strings.Contains(path, "/api/2.0/fs/files/"):
		f.handleFiles(w, r)
	default:
		http.Error(w, "unexpected request: "+r.Method+" "+path, http.StatusInternalServerError)
	}
}

func (f *fakeServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	switch {
	case r.Method == http.MethodPost && action == "initiate-upload":
		f.mu.Lock()
		f.overwriteQ = append(f.overwriteQ, r.URL.Query().Get("overwrite"))
		f.mu.Unlock()
		if f.mode == "resumable" {
			writeJSON(w, initiateResult{ResumableUpload: &uploadSession{SessionToken: "tok"}})
		} else {
			writeJSON(w, initiateResult{MultipartUpload: &uploadSession{SessionToken: "tok"}})
		}
	case r.Method == http.MethodPost && action == "complete-upload":
		var req completeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.mu.Lock()
		f.completed = req.Parts
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	case r.Method == http.MethodPut: // single-shot
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.singleBody = body
		f.overwriteQ = append(f.overwriteQ, r.URL.Query().Get("overwrite"))
		f.singleTry++
		status, respBody, hook, attempt := f.singleShotStatus, f.singleShotBody, f.singleShotHook, f.singleTry
		f.mu.Unlock()
		if hook != nil {
			if s := hook(attempt); s != 0 {
				w.WriteHeader(s)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if status != 0 {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, respBody)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "unexpected files request", http.StatusInternalServerError)
	}
}

func (f *fakeServer) handleCreatePartURLs(w http.ResponseWriter, r *http.Request) {
	var req createPartURLsRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	f.mu.Lock()
	f.partURLReqs = append(f.partURLReqs, req.Count)
	f.mu.Unlock()
	var out createPartURLsResponse
	for i := range req.Count {
		n := req.StartPartNumber + i
		out.UploadPartURLs = append(out.UploadPartURLs, struct {
			URL        string      `json:"url"`
			PartNumber int         `json:"part_number"`
			Headers    []nameValue `json:"headers"`
		}{URL: f.base() + "/cloud/part/" + strconv.Itoa(n), PartNumber: n})
	}
	writeJSON(w, out)
}

func (f *fakeServer) handlePartPut(w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/cloud/part/"))
	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	f.partHeaders = append(f.partHeaders, r.Header.Clone())
	f.partTry[n]++
	attempt := f.partTry[n]
	hook := f.partHook
	f.mu.Unlock()

	if hook != nil {
		if status, etag, respBody := hook(n, attempt); status != 0 {
			if etag != "" {
				w.Header().Set("ETag", etag)
			}
			w.WriteHeader(status)
			_, _ = w.Write(respBody)
			return
		}
	}

	f.mu.Lock()
	f.parts[n] = body
	f.mu.Unlock()
	w.Header().Set("ETag", "etag-"+strconv.Itoa(n))
	w.WriteHeader(http.StatusOK)
}

func (f *fakeServer) handleResumablePut(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	cr := r.Header.Get("Content-Range")
	// Status query: "bytes */*" -> report current confirmed offset.
	if cr == "bytes */*" {
		f.mu.Lock()
		n := len(f.resumable)
		f.mu.Unlock()
		if n == 0 {
			w.WriteHeader(http.StatusPermanentRedirect)
			return
		}
		w.Header().Set("Range", "bytes=0-"+strconv.Itoa(n-1))
		w.WriteHeader(http.StatusPermanentRedirect)
		return
	}

	// A wedged server: accept a non-final chunk but confirm no new bytes (no Range
	// header), so the client sees a zero-progress 308 and must bound its retries.
	if f.stuckResumable && strings.HasSuffix(cr, "/*") {
		w.WriteHeader(http.StatusPermanentRedirect)
		return
	}

	f.mu.Lock()
	f.resumable = append(f.resumable, body...)
	total := len(f.resumable)
	f.resumableTry++
	hook, attempt := f.resumableChunkHook, f.resumableTry
	f.mu.Unlock()

	// The chunk's bytes are committed above; a hooked transient status models a
	// blip on a chunk that actually landed, so the client's follow-up status
	// query sees forward progress.
	if hook != nil {
		if s := hook(attempt); s != 0 {
			w.WriteHeader(s)
			return
		}
	}

	// Content-Range: bytes {start}-{end}/{total|*}
	final := !strings.HasSuffix(cr, "/*")
	if final {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Range", "bytes=0-"+strconv.Itoa(total-1))
	w.WriteHeader(http.StatusPermanentRedirect)
}

// assembled returns the bytes the server reconstructed for the upload.
func (f *fakeServer) assembled() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.singleBody != nil {
		return f.singleBody
	}
	if f.mode == "resumable" {
		return f.resumable
	}
	var out []byte
	for _, p := range f.completed {
		out = append(out, f.parts[p.PartNumber]...)
	}
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func newTestClient(t *testing.T, f *fakeServer) *Client {
	t.Helper()
	return buildUploadClient(t, f.base(), f.srv.Client(), "")
}

func TestSetWorkspaceHeader(t *testing.T) {
	// A workspace-scoped client stamps the routing header the Files API control
	// plane needs; a client without a workspace ID sets nothing.
	e := newEngine(buildUploadClient(t, "https://h.test", http.DefaultClient, "1234"))
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://h.test/api/2.0/fs/files/x", nil)
	noErr(t, err)
	e.setWorkspaceHeader(req)
	if got := req.Header.Get("X-Databricks-Workspace-Id"); got != "1234" {
		t.Errorf("X-Databricks-Workspace-Id = %q, want 1234", got)
	}

	e2 := newEngine(buildUploadClient(t, "https://h.test", http.DefaultClient, ""))
	req2, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://h.test/api/2.0/fs/files/x", nil)
	noErr(t, err)
	e2.setWorkspaceHeader(req2)
	if got := req2.Header.Get("X-Databricks-Workspace-Id"); got != "" {
		t.Errorf("X-Databricks-Workspace-Id = %q, want empty", got)
	}
}

// failingSeeker reports a size (Seek to the end succeeds) but cannot rewind
// (Seek to the start fails), modeling the unrecoverable probe case.
type failingSeeker struct{ io.ReadSeeker }

func (f failingSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekStart {
		return 0, errors.New("seek start failed")
	}
	return f.ReadSeeker.Seek(offset, whence)
}

func TestUploadRewindFailureIsLoud(t *testing.T) {
	f := newFakeServer(t, "multipart")
	e := shrunkEngine(t, f)
	// A reader the engine can size but not rewind must fail the upload rather
	// than send a truncated object from the EOF position.
	r := failingSeeker{bytes.NewReader(make([]byte, 1024))}
	if _, err := e.upload(t.Context(), "/Volumes/c/s/v/f.bin", r); err == nil {
		t.Fatal("expected an error when the reader cannot be rewound after sizing")
	}
}

// testEngine returns an engine over a client pointed at f, with production
// tunables.
func testEngine(t *testing.T, f *fakeServer) *engine {
	t.Helper()
	return newEngine(newTestClient(t, f))
}

// shrunkEngine returns an engine whose size thresholds are tiny, so multipart
// boundaries are hit with small inputs. Tunables are injected per engine, so
// tests never mutate shared state and can run in parallel.
func shrunkEngine(t *testing.T, f *fakeServer) *engine {
	t.Helper()
	e := testEngine(t, f)
	e.tun.minStreamSize = 1024
	e.tun.defaultPartSize = 1024
	return e
}

// --- assertion helpers (standard library testing; no third-party framework) ---

func noErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func eqBytes(t *testing.T, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Errorf("uploaded bytes differ: got %d bytes, want %d bytes", len(got), len(want))
	}
}

func data(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 251)
	}
	return b
}

func writeTemp(t *testing.T, b []byte) string {
	t.Helper()
	p := t.TempDir() + "/src.bin"
	noErr(t, os.WriteFile(p, b, 0o600))
	return p
}

func TestUploadSingleShot(t *testing.T) {
	f := newFakeServer(t, "multipart")
	e := shrunkEngine(t, f)

	in := data(500) // below the 1024 single-shot threshold
	_, err := e.upload(t.Context(), "/Volumes/c/s/v/small.bin", bytes.NewReader(in))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestUploadMultipartParallelMatchesSequential(t *testing.T) {
	in := data(20 * 1024)

	run := func(parallelism int) []byte {
		f := newFakeServer(t, "multipart")
		e := shrunkEngine(t, f)
		_, err := e.uploadFrom(t.Context(), "/Volumes/c/s/v/big.bin", writeTemp(t, in),
			WithParallelism(parallelism))
		noErr(t, err)
		// Parts must be completed in sorted order with non-empty ETags.
		if len(f.completed) == 0 {
			t.Fatal("expected completed parts")
		}
		for i, p := range f.completed {
			if p.PartNumber != i+1 {
				t.Errorf("part %d: got part number %d, want %d", i, p.PartNumber, i+1)
			}
			if p.ETag == "" {
				t.Errorf("part %d: empty ETag", p.PartNumber)
			}
		}
		return f.assembled()
	}

	eqBytes(t, run(1), in)
	eqBytes(t, run(8), in)
}

func TestUploadFromReaderAtKnownSize(t *testing.T) {
	f := newFakeServer(t, "multipart")
	e := shrunkEngine(t, f)

	in := data(5000)
	_, err := e.upload(t.Context(), "/Volumes/c/s/v/r.bin", bytes.NewReader(in), WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestUploadNonSeekableStream(t *testing.T) {
	f := newFakeServer(t, "multipart")
	e := shrunkEngine(t, f)

	in := data(5000)
	// io.Reader wrapper that is neither Seeker nor ReaderAt.
	r := io.MultiReader(bytes.NewReader(in))
	_, err := e.upload(t.Context(), "/Volumes/c/s/v/stream.bin", r, WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestCloudPartsCarryNoDatabricksCredentials(t *testing.T) {
	f := newFakeServer(t, "multipart")
	e := shrunkEngine(t, f)

	_, err := e.uploadFrom(t.Context(), "/Volumes/c/s/v/big.bin", writeTemp(t, data(5000)), WithParallelism(4))
	noErr(t, err)

	if len(f.partHeaders) == 0 {
		t.Fatal("expected cloud part requests")
	}
	for _, h := range f.partHeaders {
		if got := h.Get("Authorization"); got != "" {
			t.Errorf("cloud part PUT must not carry Databricks auth, got Authorization=%q", got)
		}
		if got := h.Get("X-Databricks-Workspace-Id"); got != "" {
			t.Errorf("cloud part PUT must not carry workspace routing, got X-Databricks-Workspace-Id=%q", got)
		}
	}
}

func TestUploadOverwrite(t *testing.T) {
	cases := []struct {
		name string
		opts []UploadOption
		want string
	}{
		{"unset", nil, ""},
		{"true", []UploadOption{WithOverwrite(true)}, "true"},
		{"false", []UploadOption{WithOverwrite(false)}, "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeServer(t, "multipart")
			e := shrunkEngine(t, f)
			_, err := e.upload(t.Context(), "/Volumes/c/s/v/x.bin", bytes.NewReader(data(100)), tc.opts...)
			noErr(t, err)
			if len(f.overwriteQ) == 0 {
				t.Fatal("expected an overwrite query value")
			}
			if f.overwriteQ[0] != tc.want {
				t.Errorf("overwrite query = %q, want %q", f.overwriteQ[0], tc.want)
			}
		})
	}
}

func TestFileUploadMintsURLsInBatches(t *testing.T) {
	f := newFakeServer(t, "multipart")
	e := shrunkEngine(t, f)

	in := data(16 * 1024) // 16 parts at the 1024 part size
	_, err := e.uploadFrom(t.Context(), "/Volumes/c/s/v/batch.bin", writeTemp(t, in), WithParallelism(8))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)

	// The file path pre-mints in batches, so at least one create-upload-part-urls
	// call requests more than one URL; per-part minting would never exceed 1.
	maxCount := 0
	for _, cnt := range f.partURLReqs {
		maxCount = max(maxCount, cnt)
	}
	if maxCount <= 1 {
		t.Errorf("expected batched URL minting (a request with count > 1), got max count %d across %d calls", maxCount, len(f.partURLReqs))
	}
}

func TestCloudRejectsAllPartsFallsBackToSingleShot(t *testing.T) {
	f := newFakeServer(t, "multipart")
	// A cloud firewall forbids every part PUT on every attempt (a blanket block of
	// the storage host). Parts upload concurrently, so no single part is the canary;
	// because none lands, the upload falls back to single-shot.
	f.partHook = func(n, attempt int) (int, string, []byte) {
		return http.StatusForbidden, "", []byte("forbidden")
	}
	e := shrunkEngine(t, f)

	in := data(5000)
	_, err := e.uploadFrom(t.Context(), "/Volumes/c/s/v/fb.bin", writeTemp(t, in), WithParallelism(4))
	noErr(t, err)
	// The fallback single-shot PUT carries the whole object.
	eqBytes(t, f.assembled(), in)
}

func TestUploadAlreadyExists(t *testing.T) {
	f := newFakeServer(t, "multipart")
	f.singleShotStatus = http.StatusConflict
	f.singleShotBody = `{"error_code":"ALREADY_EXISTS","message":"file already exists"}`
	e := shrunkEngine(t, f)

	_, err := e.upload(t.Context(), "/Volumes/c/s/v/exists.bin", bytes.NewReader(data(100)), WithOverwrite(false))
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("got error %v, want ErrAlreadyExists", err)
	}
}

// TestWithProgressPreservesSeekable is the unit-level guard for the retry bug:
// wrapping a body for progress reporting must not hide its io.Seeker (which the
// single-shot path probes to decide retriability), yet must not fabricate one
// for a non-seekable source.
func TestWithProgressPreservesSeekable(t *testing.T) {
	p := newProgressReporter(func(Progress) {}, 10)

	// A seekable source stays seekable through the wrapper, and Seek is forwarded.
	seekable := withProgress(bytes.NewReader([]byte("0123456789")), p)
	s, ok := seekable.(io.Seeker)
	if !ok {
		t.Fatal("withProgress dropped io.Seeker for a seekable source")
	}
	if _, err := s.Seek(0, io.SeekStart); err != nil {
		t.Errorf("forwarded Seek returned error: %v", err)
	}

	// A non-seekable source must not be reported as seekable.
	if _, ok := withProgress(io.MultiReader(bytes.NewReader(nil)), p).(io.Seeker); ok {
		t.Error("withProgress fabricated io.Seeker for a non-seekable source")
	}
}

// TestSingleShotRetriesWithProgress guards both MAJOR review findings: with
// WithProgress set, a seekable single-shot body must still be retried on a
// transient failure (the progress wrapper previously hid the io.Seeker, which
// silently disabled retry).
func TestSingleShotRetriesWithProgress(t *testing.T) {
	f := newFakeServer(t, "multipart")
	// First single-shot PUT fails with a retryable 503; the second succeeds.
	f.singleShotHook = func(attempt int) int {
		if attempt == 1 {
			return http.StatusServiceUnavailable
		}
		return 0
	}
	e := shrunkEngine(t, f)
	// Shrink the control-plane backoff so the retry does not sleep a full second.
	e.tun.cpBackoff = ops.BackoffPolicy{Initial: time.Millisecond, Maximum: time.Millisecond, Factor: 1}

	var reported atomic.Int64
	in := data(500) // below the 1024 single-shot threshold
	_, err := e.upload(t.Context(), "/Volumes/c/s/v/p.bin", bytes.NewReader(in),
		WithProgress(func(p Progress) { reported.Store(p.Transferred) }))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)

	if f.singleTry != 2 {
		t.Errorf("single-shot attempts = %d, want 2 (one retry after the 503)", f.singleTry)
	}
	if reported.Load() == 0 {
		t.Error("progress callback never fired")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUploadWithTransferClient(t *testing.T) {
	f := newFakeServer(t, "multipart")
	e := shrunkEngine(t, f)

	var used atomic.Int32
	custom := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used.Add(1)
		return http.DefaultTransport.RoundTrip(r)
	})}

	in := data(5000)
	_, err := e.uploadFrom(t.Context(), "/Volumes/c/s/v/tc.bin", writeTemp(t, in),
		WithParallelism(2), WithTransferClient(custom))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
	if used.Load() == 0 {
		t.Error("the cloud transfer must go through the supplied client")
	}
}

func TestUploadResumableHappyPath(t *testing.T) {
	f := newFakeServer(t, "resumable")
	e := shrunkEngine(t, f)

	in := data(5000)
	// Parallelism > 1 still uses single-threaded resumable; bytes must match.
	_, err := e.uploadFrom(t.Context(), "/Volumes/c/s/v/gcp.bin", writeTemp(t, in), WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestUploadResumableNoProgressBounded(t *testing.T) {
	f := newFakeServer(t, "resumable")
	f.stuckResumable = true
	e := shrunkEngine(t, f)

	// The server confirms no new bytes on every chunk; the resumable loop must
	// error rather than spin forever (fs cp sets no context deadline).
	_, err := e.upload(t.Context(), "/Volumes/c/s/v/gcp.bin", bytes.NewReader(data(5000)))
	if err == nil {
		t.Fatal("expected an error when the resumable server makes no progress")
	}
}

// TestUploadResumableTransientRetryBudgetIsPerChunk guards against the retry
// budget being a global cap rather than per-progress: every chunk PUT returns a
// transient 503 (but commits its bytes, so the status query shows forward
// progress). With more chunks than maxRetries, a global budget would fail the
// upload after maxRetries total blips even though it keeps advancing; the budget
// must reset on each confirmed advance so a steadily-progressing flaky upload
// completes.
func TestUploadResumableTransientRetryBudgetIsPerChunk(t *testing.T) {
	f := newFakeServer(t, "resumable")
	e := shrunkEngine(t, f) // 1024-byte parts
	// Every data-chunk PUT reports a retryable 503 after committing; with a
	// 5000-byte input at 1024-byte chunks that is ~5 chunks, well over maxRetries.
	f.resumableChunkHook = func(attempt int) int { return http.StatusServiceUnavailable }

	in := data(5000)
	_, err := e.upload(t.Context(), "/Volumes/c/s/v/flaky.bin", bytes.NewReader(in))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
	if f.resumableTry <= e.tun.maxRetries {
		t.Errorf("test did not exceed the retry budget: %d chunk PUTs, want > maxRetries (%d)", f.resumableTry, e.tun.maxRetries)
	}
}

func TestUploadPresignedURLExpiryRetried(t *testing.T) {
	f := newFakeServer(t, "multipart")
	// Every part's first presigned URL is reported expired; the second succeeds.
	azure := `<Error><Code>AuthenticationFailed</Code><AuthenticationErrorDetail>Signature not valid in the specified time frame</AuthenticationErrorDetail></Error>`
	f.partHook = func(n, attempt int) (int, string, []byte) {
		if attempt == 1 {
			return http.StatusForbidden, "", []byte(azure)
		}
		return 0, "", nil
	}
	e := shrunkEngine(t, f)

	in := data(5000)
	_, err := e.uploadFrom(t.Context(), "/Volumes/c/s/v/exp.bin", writeTemp(t, in), WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

// Retry-timing coverage (retryable-status retried/exhausted, with shrunk backoff)
// lives in the cloudstorage package, where Send's retry policy is defined.

func TestResolvedParallelism(t *testing.T) {
	n := func(v int) *int { return &v }
	const mib int64 = 1 << 20
	tun := defaultTunables()
	tests := []struct {
		name     string
		cfg      uploadConfig
		buffered bool
		partSize int64
		want     int
	}{
		{"explicit override wins for a file", uploadConfig{parallelism: n(7)}, false, 16 * mib, 7},
		{"explicit override wins for a stream", uploadConfig{parallelism: n(3)}, true, 16 * mib, 3},
		{"file default is bandwidth-oriented", uploadConfig{}, false, 16 * mib, tun.fileParallelism},
		{"stream default fits the memory budget", uploadConfig{}, true, 16 * mib, int(tun.streamMemoryBudget / (16 * mib))},
		{"larger stream part size lowers the worker count", uploadConfig{}, true, 64 * mib, int(tun.streamMemoryBudget / (64 * mib))},
		{"tiny stream part size is capped at the file default", uploadConfig{}, true, 1, tun.fileParallelism},
		{"huge stream part size floors at one worker", uploadConfig{}, true, tun.streamMemoryBudget * 2, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.resolvedParallelism(tc.buffered, tc.partSize, tun); got != tc.want {
				t.Errorf("resolvedParallelism(%v, %d) = %d, want %d", tc.buffered, tc.partSize, got, tc.want)
			}
		})
	}
}
