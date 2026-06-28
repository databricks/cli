package upload

// Multipart upload for AWS and Azure: the part-based protocol and its
// single-threaded and parallel (producer/worker) drivers. The control-plane
// calls it relies on live in api.go; the part transfers go through uc.cloud.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/upload/cloudstorage"
	"github.com/databricks/sdk-go/core/apierr"
)

// uploadPart is the unit of work handed to an upload worker: a part number, the
// source of its bytes (an in-memory buffer for a streamed part, a section of the
// open file for a file part), and an optionally pre-minted presigned URL. When
// url is empty the worker mints one on demand; the file path pre-mints URLs in
// batches so workers go straight to the cloud PUT.
type uploadPart struct {
	partNumber int
	body       cloudstorage.Body
	url        string            // pre-minted presigned URL; "" => mint on demand
	headers    map[string]string // PUT headers accompanying url
}

// isCloudStatus reports whether err is a cloud-storage HTTP status error (an
// *apierr.APIError returned by Send) rather than a transport failure. A status
// rejection before any part has landed triggers the single-shot fallback; a
// transport failure does not.
func isCloudStatus(err error) bool {
	_, ok := errors.AsType[*apierr.APIError](err)
	return ok
}

// performMultipartUpload uploads a stream in parts on a single goroutine using
// presigned URLs (AWS/Azure).
func (c *Client) performMultipartUpload(ctx context.Context, uc *uploadContext, token string, r io.Reader, preRead []byte) error {
	currentPart := 1
	etags := map[int]string{}
	chunkOffset := int64(0)
	// AWS and Azure require each part size up front, so we buffer a part before
	// uploading it; this also lets us replay a part on retry. The buffer starts
	// with the bytes already read while deciding single-shot vs multipart.
	buffer := preRead
	retryCount := 0
	eof := false

	for !eof {
		var err error
		if buffer, err = fillBuffer(buffer, uc.partSize, r); err != nil {
			return err
		}
		if len(buffer) == 0 {
			break
		}

		urls, err := c.files.CreatePartURLs(ctx, uc.targetPath, token, currentPart, uc.batchSize)
		if err != nil {
			if chunkOffset == 0 {
				return &fallbackToFilesAPI{buffer: buffer, reason: fmt.Sprintf("failed to obtain upload URLs: %v", err)}
			}
			return err
		}

		for _, purl := range urls {
			if buffer, err = fillBuffer(buffer, uc.partSize, r); err != nil {
				return err
			}
			if len(buffer) == 0 {
				eof = true
				break
			}
			chunkLen := min(int64(len(buffer)), uc.partSize)
			headers := octetStreamHeaders(purl.Headers)

			partStart := time.Now()
			if err := uc.limiter.Acquire(ctx); err != nil {
				return err
			}
			resp, rerr := uc.cloud.Send(ctx, http.MethodPut, purl.URL, headers, cloudstorage.BytesBody(buffer[:chunkLen]))
			uc.limiter.Release()
			switch {
			case rerr == nil:
				chunkOffset += chunkLen
				etags[currentPart] = resp.Header.Get("ETag")
				buffer = buffer[chunkLen:]
				retryCount = 0
				uc.progress.add(chunkLen)
				log.Debugf(ctx, "chunk uploaded: part=%d bytes=%d offset=%d duration_ms=%d",
					currentPart, chunkLen, chunkOffset-chunkLen, time.Since(partStart).Milliseconds())
			case cloudstorage.IsURLExpired(rerr):
				if retryCount >= multipartMaxRetries {
					return errUploadURLExpired
				}
				retryCount++
				// Preserve the buffer: the same bytes are uploaded under the next
				// part number using a fresh URL.
			case chunkOffset == 0 && isCloudStatus(rerr):
				// The cloud rejected the first chunk (e.g. a firewall 403); fall back
				// to a single-shot upload through the Files API.
				return &fallbackToFilesAPI{buffer: buffer, reason: fmt.Sprintf("first chunk upload failed: %v", rerr)}
			default:
				return rerr
			}
			currentPart++
		}
	}

	return c.files.CompleteMultipart(ctx, uc.targetPath, token, etags)
}

// doUploadOnePart uploads a single part. It uses part.url when the caller
// pre-minted one (the file path mints in batches); otherwise, and whenever a URL
// expires or a slow attempt is re-issued, it mints a fresh one (count=1). While no
// part has completed yet, it converts a cloud rejection or URL-mint failure into a
// fallback signal so the caller can retry the whole upload through a single-shot
// Files API PUT (the firewall/auth case); once any part has landed, the storage
// endpoint is known good and such a failure is returned as a real error.
//
// isFirstPart marks the stream path's synchronous, uncontended first part, which
// gets its own tight soft deadline; it does not affect the fallback gating above.
func (c *Client) doUploadOnePart(ctx context.Context, uc *uploadContext, part uploadPart, token string, isFirstPart bool) (string, error) {
	start := time.Now()
	urlRetries, slowRetries := 0, 0
	url, headers := part.url, part.headers
	for {
		if url == "" {
			urls, err := c.files.CreatePartURLs(ctx, uc.targetPath, token, part.partNumber, 1)
			if err != nil {
				if !uc.completed.Load() {
					return "", &fallbackToFilesAPI{reason: fmt.Sprintf("failed to obtain upload URL for part %d: %v", part.partNumber, err)}
				}
				return "", err
			}
			url, headers = urls[0].URL, octetStreamHeaders(urls[0].Headers)
		}

		attemptStart := time.Now()
		resp, slow, rerr := c.sendPart(ctx, uc, url, headers, part.body, func() time.Duration {
			return uc.attemptDeadline(isFirstPart, slowRetries)
		})
		switch {
		case rerr == nil:
			uc.completed.Store(true)
			uc.slowGuard.record(time.Since(attemptStart))
			uc.progress.add(part.body.Size())
			log.Debugf(ctx, "part uploaded: part=%d bytes=%d first=%t duration_ms=%d",
				part.partNumber, part.body.Size(), isFirstPart, time.Since(start).Milliseconds())
			return resp.Header.Get("ETag"), nil
		case slow:
			// A wedged connection: re-mint and re-issue on a fresh URL and connection.
			// attemptDeadline disarms after slowAttemptMax tries, so a part that keeps
			// drawing slow connections rides out the normal timeouts rather than failing.
			slowRetries++
			url = ""
			log.Debugf(ctx, "part %d slow, re-issuing on a fresh connection (attempt %d)",
				part.partNumber, slowRetries+1)
		case cloudstorage.IsURLExpired(rerr):
			if urlRetries >= multipartMaxRetries {
				return "", errUploadURLExpired
			}
			urlRetries++
			url = "" // re-mint the expired URL
		case isCloudStatus(rerr) && !uc.completed.Load():
			// The cloud rejected this part before any part has landed (e.g. a firewall
			// 403 on the storage host); fall back to a single-shot upload through the
			// Files API.
			return "", &fallbackToFilesAPI{reason: fmt.Sprintf("part %d upload failed before any part completed: %v", part.partNumber, rerr)}
		default:
			return "", rerr
		}
	}
}

// sendPart performs one part PUT, cancelling it (slow=true) if it outlives the
// soft deadline so the caller can re-issue on a fresh connection (a wedged
// connection trickles bytes, escaping the idle and response timeouts). deadline
// is re-evaluated while the attempt is in flight, so a part that started under
// the cold-start deadline is caught at the tighter p95 deadline as soon as the
// guard warms up rather than waiting out the floor; deadline returning <= 0
// leaves the attempt unbounded. A ctx cancellation from the caller is reported
// as not-slow so it propagates instead of looping.
func (c *Client) sendPart(ctx context.Context, uc *uploadContext, url string, headers map[string]string, body cloudstorage.Body, deadline func() time.Duration) (resp *cloudstorage.Response, slow bool, err error) {
	// Hold one transfer slot for the duration of this attempt. A re-issued attempt
	// (slow/expiry) acquires a fresh slot, so a part never holds more than one.
	if err := uc.limiter.Acquire(ctx); err != nil {
		return nil, false, err
	}
	defer uc.limiter.Release()

	attemptCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	// The deferred cancel above fires on every return path, so attemptCtx.Done()
	// is the goroutine's stop signal once Send returns.
	start := time.Now()
	go func() {
		ticker := time.NewTicker(slowAttemptCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-attemptCtx.Done():
				return
			case <-ticker.C:
				if d := deadline(); d > 0 && time.Since(start) >= d {
					cancel(errSlowAttempt)
					return
				}
			}
		}
	}()

	resp, err = uc.cloud.Send(attemptCtx, http.MethodPut, url, headers, body)
	slow = ctx.Err() == nil && errors.Is(context.Cause(attemptCtx), errSlowAttempt)
	return resp, slow, err
}

// startPartWorkers spawns parallelism workers that drain tasks, upload each part
// from its source, and record the ETag into seed (which the caller may
// pre-populate with parts it already uploaded, e.g. the stream path's synchronous
// first part). The first failure records the error and cancels
// the rest via cancel, which a stream producer also observes to stop feeding.
// The returned wait blocks for all workers and returns the collected ETags or
// the first error. The caller owns producing and closing tasks.
func (c *Client) startPartWorkers(pctx context.Context, cancel context.CancelFunc, uc *uploadContext, token string, parallelism int, seed map[int]string, tasks <-chan uploadPart) func() (map[int]string, error) {
	results := &uploadResults{etags: seed}
	var wg sync.WaitGroup
	for range parallelism {
		wg.Go(func() {
			for part := range tasks {
				if pctx.Err() != nil {
					return
				}
				etag, e := c.doUploadOnePart(pctx, uc, part, token, false)
				if e != nil {
					results.setErr(e)
					cancel()
					return
				}
				results.put(part.partNumber, etag)
			}
		})
	}
	return func() (map[int]string, error) {
		wg.Wait()
		if err := results.err(); err != nil {
			return nil, err
		}
		return results.snapshot(), nil
	}
}

// parallelMultipartFromStream uploads a non-seekable stream in parts using a
// bounded producer/consumer. The first part is uploaded synchronously so an
// auth/firewall rejection surfaces before workers spin up and can fall back.
func (c *Client) parallelMultipartFromStream(ctx context.Context, uc *uploadContext, token string, r io.Reader) error {
	first, err := readUpTo(r, uc.partSize)
	if err != nil {
		return err
	}
	if len(first) == 0 {
		return &fallbackToFilesAPI{buffer: nil, reason: "empty input stream"}
	}
	etag1, err := c.doUploadOnePart(ctx, uc, uploadPart{partNumber: 1, body: cloudstorage.BytesBody(first)}, token, true)
	if err != nil {
		if fb, ok := errors.AsType[*fallbackToFilesAPI](err); ok {
			fb.buffer = first
			return fb
		}
		return err
	}
	if int64(len(first)) < uc.partSize {
		return c.files.CompleteMultipart(ctx, uc.targetPath, token, map[int]string{1: etag1})
	}

	pctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Hand parts straight to workers over an unbuffered channel so the bytes held
	// in memory stay bounded to the in-flight set: at most parallelism workers each
	// holding one partSize buffer, plus the one the producer is handing off. The
	// stream size is unknown, so the worker count cannot be capped by the part
	// count (as the file path does); workers beyond the number of parts observe the
	// closed channel and exit.
	tasks := make(chan uploadPart)
	wait := c.startPartWorkers(pctx, cancel, uc, token, uc.parallelism, map[int]string{1: etag1}, tasks)

	// The calling goroutine is the single producer. It reads sequential parts
	// and stops on EOF, an error, or cancellation.
	var producerErr error
	func() {
		defer close(tasks)
		partNumber := 2
		for {
			if pctx.Err() != nil {
				return
			}
			buf, rerr := readUpTo(r, uc.partSize)
			if rerr != nil {
				producerErr = rerr
				cancel()
				return
			}
			if len(buf) == 0 {
				return
			}
			select {
			case tasks <- uploadPart{partNumber: partNumber, body: cloudstorage.BytesBody(buf)}:
			case <-pctx.Done():
				return
			}
			partNumber++
			if int64(len(buf)) < uc.partSize {
				return // short read => end of stream
			}
		}
	}()

	etags, err := wait()
	if producerErr != nil {
		return producerErr
	}
	if err != nil {
		return err
	}
	return c.files.CompleteMultipart(ctx, uc.targetPath, token, etags)
}

// parallelMultipartFromReaderAt uploads a known-size, randomly-readable source in
// parts. Each part streams its own section of the shared io.ReaderAt on demand,
// so the resident bytes stay bounded to the transport's per-connection write
// buffers rather than parallelism*partSize. The caller owns the reader's
// lifetime; it must stay readable until this returns (wait() joins all workers
// first).
//
// Unlike the stream path, the first part is not uploaded synchronously: a
// randomly-readable source lets the single-shot fallback re-read from the start,
// so part 1 is just another worker task rather than a canary that blocks the
// whole upload at 0%. The fallback is instead gated on no part having completed
// (see doUploadOnePart).
func (c *Client) parallelMultipartFromReaderAt(ctx context.Context, uc *uploadContext, token string, ra io.ReaderAt) error {
	fileSize := uc.contentLength
	partSize := uc.partSize
	numParts := max(int((fileSize+partSize-1)/partSize), 1)

	pctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// A bounded channel lets the minting producer (this goroutine) run about a
	// batch ahead of the workers without holding many presigned URLs outstanding
	// at once, and gives backpressure on the slowest leg.
	tasks := make(chan uploadPart, uc.batchSize)
	wait := c.startPartWorkers(pctx, cancel, uc, token, min(uc.parallelism, numParts), map[int]string{}, tasks)

	// Mint presigned URLs in batches and hand each part to the workers pre-minted,
	// so a worker goes straight to the cloud PUT instead of minting its own URL:
	// one control-plane call per batch rather than per part, and no parallelism-wide
	// mint burst at startup. doUploadOnePart still re-mints on its own when a URL
	// expires or a slow attempt is re-issued.
	var producerErr error
	func() {
		defer close(tasks)
		for next := 1; next <= numParts; {
			if pctx.Err() != nil {
				return
			}
			count := min(uc.batchSize, numParts-next+1)
			urls, err := c.files.CreatePartURLs(pctx, uc.targetPath, token, next, count)
			if err != nil {
				if pctx.Err() != nil {
					return // cancelled by a worker failure; that error wins
				}
				if uc.completed.Load() {
					producerErr = err
				} else {
					producerErr = &fallbackToFilesAPI{reason: fmt.Sprintf("failed to obtain upload URLs at part %d: %v", next, err)}
				}
				cancel()
				return
			}
			for _, u := range urls {
				offset := int64(u.PartNumber-1) * partSize
				body := cloudstorage.SectionBody(ra, offset, min(partSize, fileSize-offset))
				select {
				case tasks <- uploadPart{partNumber: u.PartNumber, body: body, url: u.URL, headers: octetStreamHeaders(u.Headers)}:
				case <-pctx.Done():
					return
				}
			}
			next += len(urls)
		}
	}()

	etags, err := wait()
	if producerErr != nil {
		if fb, ok := errors.AsType[*fallbackToFilesAPI](producerErr); ok {
			return fb // the fallback re-reads the source, so no buffer is needed
		}
		return producerErr
	}
	if err != nil {
		if fb, ok := errors.AsType[*fallbackToFilesAPI](err); ok {
			return fb // the fallback re-reads the source, so no buffer is needed
		}
		return err
	}
	return c.files.CompleteMultipart(ctx, uc.targetPath, token, etags)
}

// uploadResults collects part ETags from workers and records the first error.
type uploadResults struct {
	mu       sync.Mutex
	etags    map[int]string
	firstErr error
}

func (r *uploadResults) put(partNumber int, etag string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.etags[partNumber] = etag
}

// setErr records the first error reported by any worker; later errors are
// dropped so the original cause survives.
func (r *uploadResults) setErr(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.firstErr == nil {
		r.firstErr = err
	}
}

func (r *uploadResults) err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.firstErr
}

func (r *uploadResults) snapshot() map[int]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return maps.Clone(r.etags)
}
