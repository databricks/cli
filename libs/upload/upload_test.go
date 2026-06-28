package upload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/upload/files"
	"github.com/databricks/databricks-sdk-go"
)

// Wire types the fake server speaks. They mirror the Files API JSON contract;
// the production copies are unexported in the files package, so the test owns
// its own copy of the shapes it fabricates and decodes.
type nameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type createPartURLsRequest struct {
	StartPartNumber int `json:"start_part_number"`
	Count           int `json:"count"`
}

type createPartURLsResponse struct {
	UploadPartURLs []struct {
		URL        string      `json:"url"`
		PartNumber int         `json:"part_number"`
		Headers    []nameValue `json:"headers"`
	} `json:"upload_part_urls"`
}

type urlWithHeaders struct {
	URL     string      `json:"url"`
	Headers []nameValue `json:"headers"`
}

type resumableURLResponse struct {
	ResumableUploadURL *urlWithHeaders `json:"resumable_upload_url"`
}

type abortURLResponse struct {
	AbortUploadURL *urlWithHeaders `json:"abort_upload_url"`
}

type uploadSession struct {
	SessionToken string `json:"session_token"`
}

type initiateResponse struct {
	MultipartUpload *uploadSession `json:"multipart_upload"`
	ResumableUpload *uploadSession `json:"resumable_upload"`
}

type completePart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type completeRequest struct {
	Parts []completePart `json:"parts"`
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

	// stuckResumable makes a resumable chunk PUT confirm no new bytes, modeling a
	// server/proxy wedged at a fixed offset.
	stuckResumable bool
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
			writeJSON(w, initiateResponse{ResumableUpload: &uploadSession{SessionToken: "tok"}})
		} else {
			writeJSON(w, initiateResponse{MultipartUpload: &uploadSession{SessionToken: "tok"}})
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
		status, respBody := f.singleShotStatus, f.singleShotBody
		f.mu.Unlock()
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
	w.Header().Set("ETag", fmt.Sprintf("etag-%d", n))
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
		w.Header().Set("Range", fmt.Sprintf("bytes=0-%d", n-1))
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
	f.mu.Unlock()

	// Content-Range: bytes {start}-{end}/{total|*}
	final := !strings.HasSuffix(cr, "/*")
	if final {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Range", fmt.Sprintf("bytes=0-%d", total-1))
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
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:     f.base(),
		Token:    "dummy-token",
		AuthType: "pat",
	})
	noErr(t, err)
	c, err := NewClient(w)
	noErr(t, err)
	return c
}

func TestControlPlaneHTTPClient(t *testing.T) {
	// A configured transport is used as-is, so the control plane honors a custom
	// CA/proxy exactly as the single-shot SDK path does, and wins over
	// InsecureSkipVerify (matching the SDK's selection order).
	custom := http.DefaultTransport.(*http.Transport).Clone()
	if c := controlPlaneHTTPClient(custom, false); c.Transport != custom {
		t.Error("a configured transport should be used as-is")
	}
	if c := controlPlaneHTTPClient(custom, true); c.Transport != custom {
		t.Error("a configured transport should win over InsecureSkipVerify")
	}

	// InsecureSkipVerify alone yields a transport that skips TLS verification.
	insecure, ok := controlPlaneHTTPClient(nil, true).Transport.(*http.Transport)
	if !ok {
		t.Fatal("InsecureSkipVerify should produce an *http.Transport")
	}
	if insecure.TLSClientConfig == nil || !insecure.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should set TLSClientConfig.InsecureSkipVerify")
	}

	// The plain default is a non-nil transport that verifies TLS.
	def, ok := controlPlaneHTTPClient(nil, false).Transport.(*http.Transport)
	if !ok {
		t.Fatal("the default should produce an *http.Transport")
	}
	if def.TLSClientConfig != nil && def.TLSClientConfig.InsecureSkipVerify {
		t.Error("the default transport must verify TLS")
	}
}

func TestFilesClientOptions(t *testing.T) {
	// A reachable host so workspace-client construction's metadata probe does not
	// stall; the response body is unused here.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "{}")
	}))
	t.Cleanup(srv.Close)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        srv.URL,
		Token:       "dummy-token",
		AuthType:    "pat",
		WorkspaceID: "1234",
	})
	noErr(t, err)

	opts := filesClientOptions(w)
	if opts.Host != srv.URL {
		t.Errorf("Host = %q, want %q", opts.Host, srv.URL)
	}
	if opts.FilesClient == nil {
		t.Error("FilesClient must be set")
	}
	if opts.HTTPClient == nil {
		t.Error("HTTPClient must be set")
	}

	// The credentials provider carries the workspace routing header and the auth
	// derived from the workspace config, which every control-plane request relies on.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/api/2.0/fs/files/x", nil)
	noErr(t, err)
	noErr(t, opts.CredentialsProvider.SetHeaders(req))
	if got := req.Header.Get("X-Databricks-Workspace-Id"); got != "1234" {
		t.Errorf("X-Databricks-Workspace-Id = %q, want 1234", got)
	}
	if req.Header.Get("Authorization") == "" {
		t.Error("Authorization header was not set")
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
	c := newTestClient(t, f)
	// A reader the engine can size but not rewind must fail the upload rather
	// than send a truncated object from the EOF position.
	r := failingSeeker{bytes.NewReader(make([]byte, 1024))}
	if _, err := c.Upload(t.Context(), "/Volumes/c/s/v/f.bin", r); err == nil {
		t.Fatal("expected an error when the reader cannot be rewound after sizing")
	}
}

// shrinkTunables makes the size thresholds tiny so multipart boundaries are hit
// with small inputs, and restores them after the test.
func shrinkTunables(t *testing.T, partSize int64) {
	t.Helper()
	origMin, origPart := multipartMinStreamSize, multipartDefaultPartSize
	multipartMinStreamSize = partSize
	multipartDefaultPartSize = partSize
	t.Cleanup(func() {
		multipartMinStreamSize = origMin
		multipartDefaultPartSize = origPart
	})
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
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	c := newTestClient(t, f)

	in := data(500) // below the 1024 single-shot threshold
	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/small.bin", bytes.NewReader(in))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestUploadMultipartParallelMatchesSequential(t *testing.T) {
	in := data(20 * 1024)

	run := func(parallelism int) []byte {
		shrinkTunables(t, 1024)
		f := newFakeServer(t, "multipart")
		c := newTestClient(t, f)
		_, err := c.UploadFrom(t.Context(), "/Volumes/c/s/v/big.bin", writeTemp(t, in),
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
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	c := newTestClient(t, f)

	in := data(5000)
	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/r.bin", bytes.NewReader(in), WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestUploadNonSeekableStream(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	c := newTestClient(t, f)

	in := data(5000)
	// io.Reader wrapper that is neither Seeker nor ReaderAt.
	r := io.MultiReader(bytes.NewReader(in))
	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/stream.bin", r, WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestCloudPartsCarryNoDatabricksCredentials(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	c := newTestClient(t, f)

	_, err := c.UploadFrom(t.Context(), "/Volumes/c/s/v/big.bin", writeTemp(t, data(5000)), WithParallelism(4))
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
			shrinkTunables(t, 1024)
			f := newFakeServer(t, "multipart")
			c := newTestClient(t, f)
			_, err := c.Upload(t.Context(), "/Volumes/c/s/v/x.bin", bytes.NewReader(data(100)), tc.opts...)
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
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	c := newTestClient(t, f)

	in := data(16 * 1024) // 16 parts at the 1024 part size
	_, err := c.UploadFrom(t.Context(), "/Volumes/c/s/v/batch.bin", writeTemp(t, in), WithParallelism(8))
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
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	// A cloud firewall forbids every part PUT on every attempt (a blanket block of
	// the storage host). Parts upload concurrently, so no single part is the canary;
	// because none lands, the upload falls back to single-shot.
	f.partHook = func(n, attempt int) (int, string, []byte) {
		return http.StatusForbidden, "", []byte("forbidden")
	}
	c := newTestClient(t, f)

	in := data(5000)
	_, err := c.UploadFrom(t.Context(), "/Volumes/c/s/v/fb.bin", writeTemp(t, in), WithParallelism(4))
	noErr(t, err)
	// The fallback single-shot PUT carries the whole object.
	eqBytes(t, f.assembled(), in)
}

func TestUploadAlreadyExists(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	f.singleShotStatus = http.StatusConflict
	f.singleShotBody = `{"error_code":"ALREADY_EXISTS","message":"file already exists"}`
	c := newTestClient(t, f)

	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/exists.bin", bytes.NewReader(data(100)), WithOverwrite(false))
	if !errors.Is(err, files.ErrAlreadyExists) {
		t.Fatalf("got error %v, want files.ErrAlreadyExists", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUploadWithTransferClient(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	c := newTestClient(t, f)

	var used atomic.Int32
	custom := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used.Add(1)
		return http.DefaultTransport.RoundTrip(r)
	})}

	in := data(5000)
	_, err := c.UploadFrom(t.Context(), "/Volumes/c/s/v/tc.bin", writeTemp(t, in),
		WithParallelism(2), WithTransferClient(custom))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
	if used.Load() == 0 {
		t.Error("the cloud transfer must go through the supplied client")
	}
}

func TestUploadResumableHappyPath(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "resumable")
	c := newTestClient(t, f)

	in := data(5000)
	// Parallelism > 1 still uses single-threaded resumable; bytes must match.
	_, err := c.UploadFrom(t.Context(), "/Volumes/c/s/v/gcp.bin", writeTemp(t, in), WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

func TestUploadResumableNoProgressBounded(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "resumable")
	f.stuckResumable = true
	c := newTestClient(t, f)

	// The server confirms no new bytes on every chunk; the resumable loop must
	// error rather than spin forever (fs cp sets no context deadline).
	_, err := c.Upload(t.Context(), "/Volumes/c/s/v/gcp.bin", bytes.NewReader(data(5000)))
	if err == nil {
		t.Fatal("expected an error when the resumable server makes no progress")
	}
}

func TestUploadPresignedURLExpiryRetried(t *testing.T) {
	shrinkTunables(t, 1024)
	f := newFakeServer(t, "multipart")
	// Every part's first presigned URL is reported expired; the second succeeds.
	azure := `<Error><Code>AuthenticationFailed</Code><AuthenticationErrorDetail>Signature not valid in the specified time frame</AuthenticationErrorDetail></Error>`
	f.partHook = func(n, attempt int) (int, string, []byte) {
		if attempt == 1 {
			return http.StatusForbidden, "", []byte(azure)
		}
		return 0, "", nil
	}
	c := newTestClient(t, f)

	in := data(5000)
	_, err := c.UploadFrom(t.Context(), "/Volumes/c/s/v/exp.bin", writeTemp(t, in), WithParallelism(4))
	noErr(t, err)
	eqBytes(t, f.assembled(), in)
}

// Retry-timing coverage (retryable-status retried/exhausted, with shrunk backoff)
// lives in the cloudstorage package, where Send's retry policy is defined.

func TestResolvedParallelism(t *testing.T) {
	n := func(v int) *int { return &v }
	const mib int64 = 1 << 20
	tests := []struct {
		name     string
		cfg      uploadConfig
		buffered bool
		partSize int64
		want     int
	}{
		{"explicit override wins for a file", uploadConfig{parallelism: n(7)}, false, 16 * mib, 7},
		{"explicit override wins for a stream", uploadConfig{parallelism: n(3)}, true, 16 * mib, 3},
		{"file default is bandwidth-oriented", uploadConfig{}, false, 16 * mib, multipartFileParallelism},
		{"stream default fits the memory budget", uploadConfig{}, true, 16 * mib, int(multipartStreamMemoryBudget / (16 * mib))},
		{"larger stream part size lowers the worker count", uploadConfig{}, true, 64 * mib, int(multipartStreamMemoryBudget / (64 * mib))},
		{"tiny stream part size is capped at the file default", uploadConfig{}, true, 1, multipartFileParallelism},
		{"huge stream part size floors at one worker", uploadConfig{}, true, multipartStreamMemoryBudget * 2, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.resolvedParallelism(tc.buffered, tc.partSize); got != tc.want {
				t.Errorf("resolvedParallelism(%v, %d) = %d, want %d", tc.buffered, tc.partSize, got, tc.want)
			}
		})
	}
}
