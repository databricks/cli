package upload

// Upload-side glue between the orchestration and the two transports: the
// single-shot delegation to the authenticated files.Client, and the best-effort
// aborts (mint a presigned abort URL on the control plane, then issue the
// unauthenticated cloud DELETE).

import (
	"context"
	"io"
	"net/http"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/upload/files"
)

// singleShotUpload sends the whole body in one PUT through the files.Client.
// Used for small files and as the multipart/resumable fallback. Progress is
// reported as the body streams; the files.Client itself is progress-agnostic.
func (c *Client) singleShotUpload(ctx context.Context, uc *uploadContext, body io.Reader) error {
	if uc.progress != nil {
		body = &progressReader{r: body, p: uc.progress}
	}
	// A single-shot upload is one transfer; hold one slot for its duration so it
	// draws from the same concurrency budget as multipart parts.
	if err := uc.limiter.Acquire(ctx); err != nil {
		return err
	}
	defer uc.limiter.Release()
	return c.files.Upload(ctx, uc.targetPath, uc.overwrite, body)
}

// abortMultipartUpload mints a presigned abort URL on the control plane and
// issues the unauthenticated cloud DELETE against it. Both halves run on the
// caller's context, which abortMultipartBestEffort detaches and bounds.
func (c *Client) abortMultipartUpload(ctx context.Context, uc *uploadContext, token string) error {
	purl, err := c.files.CreateAbortURL(ctx, uc.targetPath, token)
	if err != nil {
		return err
	}
	headers := octetStreamHeaders(purl.Headers)
	_, err = uc.cloud.Send(ctx, http.MethodDelete, purl.URL, headers, nil)
	return err
}

func (c *Client) abortMultipartBestEffort(ctx context.Context, uc *uploadContext, token string) {
	ctx, cancel := cleanupContext(ctx)
	defer cancel()
	if err := c.abortMultipartUpload(ctx, uc, token); err != nil {
		// Best-effort cleanup; its failure is not user-actionable (e.g. some clouds
		// do not support abort presigned URLs), so it stays at debug rather than
		// surfacing as a warning on an otherwise-normal outcome like a skip.
		log.Debugf(ctx, "failed to abort multipart upload: %v", err)
	}
}

// cleanupContext returns a context for a best-effort cleanup (aborting a partial
// upload) that is detached from the caller's cancellation and deadline but
// bounded by its own timeout. Cleanup most needs to run exactly when the upload
// context has already been cancelled or has expired; reusing it would make the
// abort fail instantly and leak the partial upload on the storage provider.
func cleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), cloudCleanupTimeout)
}

func (c *Client) abortResumableUpload(ctx context.Context, uc *uploadContext, purl files.PresignedURL) error {
	ctx, cancel := cleanupContext(ctx)
	defer cancel()
	_, err := uc.cloud.Send(ctx, http.MethodDelete, purl.URL, purl.Headers, nil)
	return err
}
