// Package upload uploads large files to Databricks Unity Catalog Volumes,
// choosing a single-shot, multipart (AWS/Azure), or resumable (GCP) protocol
// based on the stream size and the protocol the workspace's Files API selects.
//
// It is a CLI-agnostic port of the Databricks Go SDK files/v2 ext_upload mixin.
// The package is pure orchestration over two transports: the authenticated
// Databricks Files API client (the files package), which handles the single-shot
// upload and the multipart/resumable control plane, and the unauthenticated
// cloud-storage client (the cloudstorage package), which transfers parts and chunks
// directly to object storage. Callers pass a [*databricks.WorkspaceClient]; see
// [NewClient].
package upload

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/upload/cloudstorage"
	"github.com/databricks/cli/libs/upload/files"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config/credentials"
)

// Tunables that mirror the SDK defaults. They are package variables (not
// constants) so tests can shrink them to keep boundary cases fast; production
// code never mutates them.
var (
	// multipartMinStreamSize is the threshold below which a known-size upload
	// is sent in a single PUT instead of being split into parts.
	multipartMinStreamSize int64 = 50 * 1024 * 1024

	// multipartDefaultPartSize is the part size used when no part size is
	// requested. A fixed, modestly sized part (rather than one scaled to keep the
	// part count low) bounds the memory a buffering upload holds (about
	// parallelism*partSize for a non-seekable stream) and spreads stragglers
	// across more parts. It is only grown when a file is large enough to exceed
	// maxUploadParts.
	multipartDefaultPartSize int64 = 16 << 20 // 16 MiB

	// multipartMaxPartSize is the largest part size a cloud provider accepts.
	multipartMaxPartSize int64 = 4 << 30

	// multipartBatchURLCount is the number of presigned URLs requested per batch
	// when the content length is unknown.
	multipartBatchURLCount = 1

	// multipartFileParallelism is the default worker count when no parallelism is
	// requested and the source is randomly readable (a local file or other
	// io.ReaderAt). Each part streams from a positioned read, so resident memory
	// stays bounded to the transport's per-connection buffers regardless of the
	// worker count (not parallelism*partSize); a high default saturates a fast
	// uplink, and the driver caps the workers at the part count for small files.
	multipartFileParallelism = 128

	// multipartStreamMemoryBudget bounds the bytes a buffered upload (a
	// non-seekable stream) holds in memory, where each worker keeps one partSize
	// buffer for its in-flight part. The default stream worker count is this budget
	// divided by the part size, so a larger part size lowers the worker count
	// instead of growing memory. At the 16 MiB default part size this is 16 workers.
	multipartStreamMemoryBudget int64 = 256 << 20

	// multipartReadAheadBytes is how far past a chunk the resumable upload reads
	// to detect end-of-stream (a resumable chunk cannot be empty, so the last
	// chunk must be marked with the real total size).
	multipartReadAheadBytes int64 = 1

	// multipartMaxRetries caps how many times a single part is retried after a
	// presigned URL expires (multipart) or a chunk must be resumed (resumable).
	multipartMaxRetries = 3

	// maxUploadParts is the provider's maximum number of parts per multipart
	// upload (S3's 10,000 is the common floor). The default part size is grown
	// for files large enough to otherwise exceed it.
	maxUploadParts int64 = 10_000
)

// Cloud-transport tunables for the unauthenticated part/chunk transfers.
var (
	// cloudResponseHeaderTimeout bounds the response phase of a cloud transfer so
	// a connection that accepts the upload but never replies cannot hang a
	// goroutine forever. It is applied to the transfer client built in
	// resolveCloudClient.
	cloudResponseHeaderTimeout = 60 * time.Second

	// cloudCleanupTimeout bounds a best-effort abort, which runs on a context
	// detached from the upload's (see cleanupContext).
	cloudCleanupTimeout = 30 * time.Second
)

// errUploadURLExpired is returned when a part's presigned URL keeps expiring
// across re-mint retries. The unexpected-response and already-exists conditions
// are owned by the files package ([files.ErrUnexpectedServerResponse],
// [files.ErrAlreadyExists]).
var errUploadURLExpired = errors.New("upload URL expired after retries")

// fallbackToFilesAPI signals that a multipart or resumable upload failed in a
// way that warrants retrying through a single-shot Files API PUT (for example,
// a cloud firewall rejecting the first part). It carries the bytes already
// buffered but not yet uploaded so they can be replayed.
type fallbackToFilesAPI struct {
	buffer []byte
	reason string
}

func (e *fallbackToFilesAPI) Error() string {
	return "falling back to single-shot upload: " + e.reason
}

// Client uploads files to Unity Catalog Volumes. It is pure orchestration: the
// single-shot upload and the multipart/resumable control plane run through the
// authenticated [files.Client]; the part/chunk transfers run over an
// unauthenticated cloud client built per upload (see resolveCloudClient).
type Client struct {
	files *files.Client
}

// NewClient returns a Client that uploads through w's Files API.
func NewClient(w *databricks.WorkspaceClient) (*Client, error) {
	if w == nil {
		return nil, errors.New("workspace client must not be nil")
	}
	fc, err := files.New(filesClientOptions(w))
	if err != nil {
		return nil, err
	}
	return &Client{files: fc}, nil
}

// filesClientOptions extracts the Files API client options from a workspace
// client: the SDK Files client for single-shot uploads, the control-plane host,
// per-request credentials (workspace routing headers plus auth), and an HTTP
// client whose transport honors the workspace config.
func filesClientOptions(w *databricks.WorkspaceClient) files.ClientOptions {
	// The control-plane requests are authenticated and routed via the workspace
	// config; the cloud part transfers are unauthenticated and never see these.
	creds := credentials.CredentialsProviderFn(func(r *http.Request) error {
		for k, v := range auth.WorkspaceIDHeaders(w.Config) {
			r.Header.Set(k, v)
		}
		return w.Config.Authenticate(r)
	})
	return files.ClientOptions{
		FilesClient:         w.Files,
		Host:                w.Config.Host,
		CredentialsProvider: creds,
		HTTPClient:          controlPlaneHTTPClient(w.Config.HTTPTransport, w.Config.InsecureSkipVerify),
	}
}

// controlPlaneHTTPClient builds the HTTP client for the Files API control plane.
// It mirrors the SDK's transport selection (httpclient.ClientConfig.httpTransport)
// so a custom CA, proxy, or InsecureSkipVerify configured on the workspace is
// honored exactly as the single-shot path (w.Files) honors it. Credentials are
// applied per request, so the client carries no auth of its own.
func controlPlaneHTTPClient(custom http.RoundTripper, insecureSkipVerify bool) *http.Client {
	switch {
	case custom != nil:
		return &http.Client{Transport: custom}
	case insecureSkipVerify:
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		return &http.Client{Transport: t}
	default:
		return &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()}
	}
}

// UploadResult holds the result of an upload. It is currently empty and exists
// for forward compatibility.
type UploadResult struct{}

// UploadOption configures an Upload or UploadFrom call.
type UploadOption func(*uploadConfig)

type uploadConfig struct {
	overwrite      *bool
	partSize       int64
	parallelism    *int
	progress       ProgressFunc
	transferClient *http.Client
	limiter        Limiter
}

// Progress reports the state of an in-flight upload. Fields may be added in
// future releases, so callers must not depend on the struct being comparable or
// on its exact size; always refer to fields by name.
type Progress struct {
	// Transferred is the cumulative number of bytes confirmed uploaded so far.
	Transferred int64
	// Total is the total size in bytes, or -1 if it is not known in advance (a
	// non-seekable stream).
	Total int64
}

// ProgressFunc is invoked as an upload makes progress. It is called from
// internal goroutines but never concurrently with itself, so it needs no
// locking of its own; it must return promptly.
type ProgressFunc func(Progress)

// WithOverwrite controls whether an existing file is overwritten. When this
// option is not supplied the parameter is omitted and the server applies its
// default.
func WithOverwrite(overwrite bool) UploadOption {
	return func(c *uploadConfig) { c.overwrite = &overwrite }
}

// WithPartSize sets the multipart part size in bytes. It must not exceed the
// cloud provider maximum. When not supplied an appropriate size is chosen from
// the content length.
//
//deadcode:allow exported engine option with no in-tree caller yet
func WithPartSize(partSize int64) UploadOption {
	return func(c *uploadConfig) { c.partSize = partSize }
}

// WithParallelism sets the number of concurrent upload workers used for a large
// file. A value of 1 uploads the parts sequentially on a single goroutine;
// higher values upload that many parts at once. It must be at least 1. When not
// set, a default is used.
func WithParallelism(n int) UploadOption {
	return func(c *uploadConfig) { c.parallelism = &n }
}

// WithProgress registers a callback invoked as the upload progresses, reporting
// the cumulative bytes uploaded and the total size (-1 if unknown). It is useful
// for rendering an upload progress bar.
func WithProgress(fn ProgressFunc) UploadOption {
	return func(c *uploadConfig) { c.progress = fn }
}

// WithTransferClient overrides the HTTP client used to transfer file contents
// during a large-file upload, replacing the one the call would otherwise create
// internally. Use it to route the transfer through a custom transport, proxy, or
// CA.
//
// Because these transfers use self-authenticating presigned URLs, the client must
// not attach Databricks credentials; the storage provider rejects extra
// authentication. Do not set http.Client.Timeout, a whole-request deadline that
// would abort a legitimately long part transfer; bound the upload with the context
// passed to Upload instead.
func WithTransferClient(client *http.Client) UploadOption {
	return func(c *uploadConfig) { c.transferClient = client }
}

func resolveUploadConfig(opts []UploadOption) (*uploadConfig, error) {
	cfg := &uploadConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.parallelism != nil && *cfg.parallelism < 1 {
		return nil, fmt.Errorf("parallelism must be at least 1, got %d", *cfg.parallelism)
	}
	if cfg.partSize < 0 {
		return nil, fmt.Errorf("part size must be non-negative, got %d", cfg.partSize)
	}
	if cfg.partSize > multipartMaxPartSize {
		return nil, fmt.Errorf("part size %d exceeds maximum %d", cfg.partSize, multipartMaxPartSize)
	}
	return cfg, nil
}

// resolvedParallelism returns the worker count: the caller's explicit value when
// set, otherwise a default that depends on whether the source is buffered. A
// randomly-readable source streams parts from positioned reads and uses a high,
// bandwidth-oriented default; a buffered (non-seekable) source holds
// parallelism*partSize in memory, so its default is sized to a memory budget,
// capped at the file default so a tiny part size cannot translate the budget into
// an unbounded number of goroutines and sockets.
func (cfg *uploadConfig) resolvedParallelism(buffered bool, partSize int64) int {
	if cfg.parallelism != nil {
		return *cfg.parallelism
	}
	if !buffered {
		return multipartFileParallelism
	}
	workers := int(multipartStreamMemoryBudget / partSize)
	return min(max(workers, 1), multipartFileParallelism)
}

// NewTransferClient returns an HTTP client suitable for WithTransferClient, sized
// for n concurrent transfers: it keeps up to n idle connections to the storage
// host so they are reused rather than re-dialed (Go's default of 2 per host would
// otherwise force re-dialing). Share one across uploads that also share a Limiter
// so their connection pool is shared and reused. It attaches no Databricks
// credentials (presigned URLs are self-authenticating) and sets no whole-request
// timeout, which would abort a legitimately long transfer; bound the upload with
// the context instead.
func NewTransferClient(n int) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = cloudResponseHeaderTimeout
	transport.MaxIdleConnsPerHost = n
	transport.MaxIdleConns = max(transport.MaxIdleConns, n)
	return &http.Client{Transport: transport}
}

// resolveCloudClient returns the caller-supplied transfer client when set, and
// otherwise a fresh one sized to the upload parallelism.
func (cfg *uploadConfig) resolveCloudClient(parallelism int) *http.Client {
	if cfg.transferClient != nil {
		return cfg.transferClient
	}
	return NewTransferClient(parallelism)
}

// uploadContext is the resolved per-upload state. Its fields are set once at
// construction; the slowGuard and completed flag hold the only mutable state,
// both safe for concurrent use by the upload workers.
type uploadContext struct {
	targetPath    string
	overwrite     *bool
	partSize      int64
	batchSize     int
	contentLength int64 // -1 when unknown (non-seekable stream)
	sourcePath    string
	parallel      bool
	parallelism   int
	cloud         *cloudstorage.Client
	progress      *progressReporter
	slowGuard     *slowAttemptGuard
	limiter       Limiter // bounds concurrent transfers; never nil (unlimited by default)

	// completed records whether any part has finished uploading. The single-shot
	// fallback is taken only while no part has landed; once a part succeeds, a
	// later cloud rejection is a real error rather than a blanket-block signal.
	completed atomic.Bool
}

// newUploadContext resolves the per-upload state shared by Upload and UploadFrom.
// contentLength is -1 for a non-seekable stream; sourcePath is "" unless the
// source is a local file (UploadFrom). buffered reports whether the upload path
// holds parts in memory (a non-seekable stream) rather than streaming them from
// positioned reads, which sets the default worker count.
func newUploadContext(cfg *uploadConfig, targetPath, sourcePath string, contentLength int64, buffered bool) *uploadContext {
	partSize, batchSize := optimizeParams(contentLength, cfg.partSize)
	parallelism := cfg.resolvedParallelism(buffered, partSize)
	limiter := cfg.limiter
	if limiter == nil {
		limiter = unlimitedLimiter{}
	}
	return &uploadContext{
		targetPath:    targetPath,
		overwrite:     cfg.overwrite,
		partSize:      partSize,
		batchSize:     batchSize,
		contentLength: contentLength,
		sourcePath:    sourcePath,
		parallel:      parallelism > 1,
		parallelism:   parallelism,
		cloud:         cloudstorage.New(cfg.resolveCloudClient(parallelism)),
		progress:      newProgressReporter(cfg.progress, contentLength),
		slowGuard:     &slowAttemptGuard{},
		limiter:       limiter,
	}
}

// progressReporter accumulates confirmed-uploaded bytes and forwards the running
// total to a ProgressFunc. Its methods are safe for concurrent use (parallel
// uploads report from many goroutines) and serialize callbacks so the
// ProgressFunc never runs concurrently with itself.
type progressReporter struct {
	fn    ProgressFunc
	total int64

	mu          sync.Mutex
	transferred int64
}

// newProgressReporter returns nil when fn is nil so that reporter calls are
// no-ops without the callers having to check.
func newProgressReporter(fn ProgressFunc, total int64) *progressReporter {
	if fn == nil {
		return nil
	}
	return &progressReporter{fn: fn, total: total}
}

// add records n newly uploaded bytes and reports the new cumulative total. It is
// a no-op on a nil reporter, so callers need not check.
func (p *progressReporter) add(n int64) {
	if p == nil || n <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transferred += n
	p.fn(Progress{Transferred: p.transferred, Total: p.total})
}

// progressReader reports bytes as they are read. It tracks single-shot uploads,
// where the whole body streams through one request rather than discrete parts.
type progressReader struct {
	r io.Reader
	p *progressReporter
}

func (pr *progressReader) Read(b []byte) (int, error) {
	n, err := pr.r.Read(b)
	pr.p.add(int64(n))
	return n, err
}

// Upload uploads contents to the remote file at filePath, which must be an
// absolute path such as "/Volumes/catalog/schema/volume/file". It chooses a
// single-shot, multipart, or resumable upload based on the stream size (when
// the reader is seekable) and the protocol the workspace's Files API selects.
//
// When contents implements io.ReaderAt (for example an *os.File or a
// *bytes.Reader), parts are read from it with concurrent positioned reads rather
// than buffered in memory. A reader without random access has each in-flight part
// buffered as it is read.
//
// ctx bounds the entire operation, including all retries; pass a context with a
// deadline to cap the total upload time.
func (c *Client) Upload(ctx context.Context, filePath string, contents io.Reader, opts ...UploadOption) (*UploadResult, error) {
	cfg, err := resolveUploadConfig(opts)
	if err != nil {
		return nil, err
	}

	// A seekable reader lets us learn the size up front; otherwise it is unknown.
	// Seeking to the end moves the reader, so a failed rewind afterward is
	// unrecoverable: the reader is left at EOF and a later read would send zero
	// bytes. Fail loud in that case rather than silently truncating the object.
	contentLength := int64(-1)
	if seeker, ok := contents.(io.Seeker); ok {
		if end, serr := seeker.Seek(0, io.SeekEnd); serr == nil {
			if _, serr := seeker.Seek(0, io.SeekStart); serr != nil {
				return nil, fmt.Errorf("cannot rewind reader after sizing: %w", serr)
			}
			contentLength = end
		}
	}

	// A randomly-readable source large enough for the parallel path streams its
	// parts from positioned reads; any other source buffers each in-flight part.
	_, isReaderAt := contents.(io.ReaderAt)
	buffered := !isReaderAt || contentLength < multipartMinStreamSize
	uc := newUploadContext(cfg, filePath, "", contentLength, buffered)

	switch {
	case uc.parallel && contentLength < 0:
		// Non-seekable stream of unknown size: parts are buffered as they are read.
		return &UploadResult{}, c.parallelUploadFromStream(ctx, uc, contents)
	case uc.parallel && contentLength >= multipartMinStreamSize:
		// Known-size large upload. A reader supporting concurrent positioned reads
		// (io.ReaderAt) is streamed in sections without buffering; anything else
		// buffers parts.
		if ra, ok := contents.(io.ReaderAt); ok {
			return &UploadResult{}, c.parallelUploadFromReaderAt(ctx, uc, ra)
		}
		return &UploadResult{}, c.parallelUploadFromStream(ctx, uc, contents)
	case contentLength >= 0:
		return &UploadResult{}, c.uploadSingleThreadKnownSize(ctx, uc, contents)
	default:
		return &UploadResult{}, c.singleThreadMultipart(ctx, uc, contents)
	}
}

// UploadFrom uploads the local file at sourcePath to the remote file at
// filePath. The local file is opened as needed; its size is always known, so it
// never takes the non-seekable path.
//
// Parts are read from the file on demand rather than buffered, so the upload
// covers the file as of its size when the call begins: bytes appended afterward
// are not included, and truncating the file mid-upload fails the upload (it does
// not produce a partial object). Do not modify the file while an upload is in
// progress.
//
// ctx bounds the entire operation, including all retries; pass a context with a
// deadline to cap the total upload time.
func (c *Client) UploadFrom(ctx context.Context, filePath, sourcePath string, opts ...UploadOption) (*UploadResult, error) {
	cfg, err := resolveUploadConfig(opts)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	size := info.Size()

	// A local file is randomly readable, so the parallel path streams parts from
	// positioned reads and never buffers them: buffered is false.
	uc := newUploadContext(cfg, filePath, sourcePath, size, false)

	if uc.parallel && size >= multipartMinStreamSize {
		return &UploadResult{}, c.parallelUploadFromFile(ctx, uc)
	}
	f, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return &UploadResult{}, c.uploadSingleThreadKnownSize(ctx, uc, f)
}

// optimizeParams picks a part size and a presigned-URL batch size. contentLength
// is -1 when unknown; override is 0 when no part size was requested. The override
// is already range-checked in resolveUploadConfig.
func optimizeParams(contentLength, override int64) (partSize int64, batchSize int) {
	switch {
	case override > 0:
		partSize = override
	default:
		partSize = multipartDefaultPartSize
		// Grow the part size only if the fixed default would exceed the
		// provider's maximum part count for this (known) content length.
		if contentLength >= 0 {
			if minPart := (contentLength + maxUploadParts - 1) / maxUploadParts; minPart > partSize {
				partSize = min(minPart, multipartMaxPartSize)
			}
		}
	}

	batchSize = multipartBatchURLCount
	if contentLength >= 0 && partSize > 0 {
		numParts := (contentLength + partSize - 1) / partSize
		// The square root of the part count is a heuristic that balances the
		// number of URL-minting round trips against batch size.
		batchSize = max(int(math.Ceil(math.Sqrt(float64(numParts)))), 1)
	}
	return partSize, batchSize
}

func (c *Client) uploadSingleThreadKnownSize(ctx context.Context, uc *uploadContext, r io.Reader) error {
	if uc.contentLength < multipartMinStreamSize {
		return c.singleShotUpload(ctx, uc, r)
	}
	return c.singleThreadMultipart(ctx, uc, r)
}

// singleThreadMultipart pre-reads up to the single-shot threshold to handle
// small (or non-seekable but actually small) streams without a multipart
// round trip, then initiates the upload.
func (c *Client) singleThreadMultipart(ctx context.Context, uc *uploadContext, r io.Reader) error {
	preRead, err := readUpTo(r, multipartMinStreamSize)
	if err != nil {
		return err
	}
	if int64(len(preRead)) < multipartMinStreamSize {
		return c.singleShotUpload(ctx, uc, bytes.NewReader(preRead))
	}
	return c.initiateAndUpload(ctx, uc, false,
		func(token string) error { return c.performMultipartUpload(ctx, uc, token, r, preRead) },
		func(token string) error { return c.performResumableUpload(ctx, uc, token, r, preRead) },
		func(buffered []byte) error {
			return c.singleShotUpload(ctx, uc, io.MultiReader(bytes.NewReader(buffered), r))
		},
	)
}

func (c *Client) parallelUploadFromStream(ctx context.Context, uc *uploadContext, r io.Reader) error {
	return c.initiateAndUpload(ctx, uc, true,
		func(token string) error { return c.parallelMultipartFromStream(ctx, uc, token, r) },
		func(token string) error { return c.performResumableUpload(ctx, uc, token, r, nil) },
		func(buffered []byte) error {
			return c.singleShotUpload(ctx, uc, io.MultiReader(bytes.NewReader(buffered), r))
		},
	)
}

func (c *Client) parallelUploadFromFile(ctx context.Context, uc *uploadContext) error {
	// Each branch opens its own handle: only one runs per upload, and the multipart
	// driver keeps its handle open until all its workers finish (wait()).
	withFile := func(use func(*os.File) error) error {
		f, err := os.Open(uc.sourcePath)
		if err != nil {
			return err
		}
		defer f.Close()
		return use(f)
	}
	return c.initiateAndUpload(ctx, uc, true,
		func(token string) error {
			return withFile(func(f *os.File) error { return c.parallelMultipartFromReaderAt(ctx, uc, token, f) })
		},
		func(token string) error {
			return withFile(func(f *os.File) error { return c.performResumableUpload(ctx, uc, token, f, nil) })
		},
		func(buffered []byte) error {
			return withFile(func(f *os.File) error { return c.singleShotUpload(ctx, uc, f) })
		},
	)
}

// parallelUploadFromReaderAt uploads a known-size, randomly-readable source
// (an *os.File or *bytes.Reader passed to Upload) by streaming sections of it
// concurrently, without buffering whole parts. The resumable and fallback paths
// re-read it through fresh section readers.
func (c *Client) parallelUploadFromReaderAt(ctx context.Context, uc *uploadContext, ra io.ReaderAt) error {
	return c.initiateAndUpload(ctx, uc, true,
		func(token string) error { return c.parallelMultipartFromReaderAt(ctx, uc, token, ra) },
		func(token string) error {
			return c.performResumableUpload(ctx, uc, token, io.NewSectionReader(ra, 0, uc.contentLength), nil)
		},
		func(buffered []byte) error {
			return c.singleShotUpload(ctx, uc, io.NewSectionReader(ra, 0, uc.contentLength))
		},
	)
}

// initiateAndUpload initiates the server-coordinated upload, dispatches to the
// multipart or resumable path the server selected (multipart on AWS/Azure,
// resumable on GCP), and applies the shared abort-and-fallback handling.
func (c *Client) initiateAndUpload(
	ctx context.Context,
	uc *uploadContext,
	parallel bool,
	performMultipart func(token string) error,
	performResumable func(token string) error,
	fallbackSingleShot func(buffered []byte) error,
) error {
	resp, err := c.files.Initiate(ctx, uc.targetPath, uc.overwrite)
	if err != nil {
		return err
	}

	switch {
	case resp.MultipartUpload != nil:
		token := resp.MultipartUpload.SessionToken
		if token == "" {
			return fmt.Errorf("%w: missing multipart session token", files.ErrUnexpectedServerResponse)
		}
		uerr := performMultipart(token)
		if uerr == nil {
			return nil
		}
		// Abort is best-effort in both the fallback and the hard-error case.
		c.abortMultipartBestEffort(ctx, uc, token)
		if fb, ok := errors.AsType[*fallbackToFilesAPI](uerr); ok {
			log.Infof(ctx, "using single-shot fallback: %s", fb.reason)
			return fallbackSingleShot(fb.buffer)
		}
		return uerr

	case resp.ResumableUpload != nil:
		token := resp.ResumableUpload.SessionToken
		if token == "" {
			return fmt.Errorf("%w: missing resumable session token", files.ErrUnexpectedServerResponse)
		}
		if parallel {
			log.Warnf(ctx, "GCP does not support parallel resumable uploads; using single-threaded upload")
		}
		// The resumable path aborts itself on error, so no abort here.
		uerr := performResumable(token)
		if fb, ok := errors.AsType[*fallbackToFilesAPI](uerr); ok {
			log.Infof(ctx, "using single-shot fallback: %s", fb.reason)
			return fallbackSingleShot(fb.buffer)
		}
		return uerr

	default:
		return fmt.Errorf("%w: neither multipart nor resumable upload offered", files.ErrUnexpectedServerResponse)
	}
}
