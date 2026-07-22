package files

// Shared buffering and cloud-leg header helpers for the large-file uploader,
// plus the best-effort abort glue between the orchestration and the two
// transports. The cloud-transport pieces (part bodies, retry classification,
// presigned-URL-expiry detection) live in the cloudstorage subpackage.

import (
	"context"
	"io"
	"maps"
	"net/http"
)

// --- Buffer helpers ---

// fillBuffer reads from r until buf holds at least minSize bytes or the stream
// ends. A short read at end-of-stream is not an error.
func fillBuffer(buf []byte, minSize int64, r io.Reader) ([]byte, error) {
	need := minSize - int64(len(buf))
	if need <= 0 {
		return buf, nil
	}
	tmp := make([]byte, need)
	n, err := io.ReadFull(r, tmp)
	buf = append(buf, tmp[:n]...)
	if err == nil || err == io.EOF || err == io.ErrUnexpectedEOF {
		return buf, nil
	}
	return buf, err
}

// readUpTo reads up to n bytes from r, returning fewer only at end-of-stream.
func readUpTo(r io.Reader, n int64) ([]byte, error) {
	buf := make([]byte, n)
	read, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return buf[:read], nil
}

// --- Header helpers ---

func mergeHeaders(base, override map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(override))
	maps.Copy(out, base)
	maps.Copy(out, override)
	return out
}

// octetStreamHeaders returns the headers for a cloud-storage request: the binary
// content type plus the presigned URL's own headers (which win on conflict). The
// returned map is freshly allocated, so callers may add request-specific headers
// such as Content-Range.
func octetStreamHeaders(presigned map[string]string) map[string]string {
	return mergeHeaders(map[string]string{"Content-Type": "application/octet-stream"}, presigned)
}

// --- Single-shot delegation and best-effort abort glue ---

// singleShotUpload sends the whole body in one PUT through the control-plane
// single-shot path. Used for small files and as the multipart/resumable
// fallback. Progress is reported as the body streams.
func (e *engine) singleShotUpload(ctx context.Context, uc *uploadContext, body io.Reader) error {
	if uc.progress != nil {
		body = withProgress(body, uc.progress)
	}
	// A single-shot upload is one transfer; hold one slot for its duration so it
	// draws from the same concurrency budget as multipart parts.
	if err := uc.limiter.Acquire(ctx); err != nil {
		return err
	}
	defer uc.limiter.Release()
	return e.uploadSingleShot(ctx, uc.targetPath, uc.overwrite, body)
}

// abortMultipartUpload mints a presigned abort URL on the control plane and
// issues the unauthenticated cloud DELETE against it. Both halves run on the
// caller's context, which abortMultipartBestEffort detaches and bounds.
func (e *engine) abortMultipartUpload(ctx context.Context, uc *uploadContext, token string) error {
	purl, err := e.createAbortURL(ctx, uc.targetPath, token)
	if err != nil {
		return err
	}
	headers := octetStreamHeaders(purl.Headers)
	_, err = uc.cloud.Send(ctx, http.MethodDelete, purl.URL, headers, nil)
	return err
}

func (e *engine) abortMultipartBestEffort(ctx context.Context, uc *uploadContext, token string) {
	ctx, cancel := e.cleanupContext(ctx)
	defer cancel()
	if err := e.abortMultipartUpload(ctx, uc, token); err != nil {
		// Best-effort cleanup; its failure is not user-actionable (e.g. some clouds
		// do not support abort presigned URLs), so it stays at debug rather than
		// surfacing as a warning on an otherwise-normal outcome like a skip.
		e.c.logger.DebugContext(ctx, "failed to abort multipart upload", "error", err)
	}
}

// cleanupContext returns a context for a best-effort cleanup (aborting a partial
// upload) that is detached from the caller's cancellation and deadline but
// bounded by its own timeout. Cleanup most needs to run exactly when the upload
// context has already been cancelled or has expired; reusing it would make the
// abort fail instantly and leak the partial upload on the storage provider.
func (e *engine) cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), e.tun.cleanupTimeout)
}

func (e *engine) abortResumableUpload(ctx context.Context, uc *uploadContext, purl presignedURL) error {
	ctx, cancel := e.cleanupContext(ctx)
	defer cancel()
	_, err := uc.cloud.Send(ctx, http.MethodDelete, purl.URL, purl.Headers, nil)
	return err
}
