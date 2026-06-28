package upload

// Resumable upload for GCP: the single-stream, offset-resuming protocol used
// when the server hands back a resumable session instead of multipart. Chunk
// transfers go through uc.cloud.Attempt (raw responses, since 308 is the normal
// "continue" signal); the loop runs its own resume-aware retry.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/upload/cloudstorage"
	"github.com/databricks/cli/libs/upload/files"
	"github.com/databricks/sdk-go/core/apierr"
)

var resumableRangeRe = regexp.MustCompile(`bytes=0-(\d+)`)

// performResumableUpload uploads a stream using the GCP resumable protocol,
// aborting the session if any chunk fails.
func (c *Client) performResumableUpload(ctx context.Context, uc *uploadContext, token string, r io.Reader, preRead []byte) error {
	purl, err := c.files.CreateResumableURL(ctx, uc.targetPath, token)
	if err != nil {
		return &fallbackToFilesAPI{buffer: preRead, reason: fmt.Sprintf("failed to obtain resumable upload URL: %v", err)}
	}
	if err := c.resumableUploadLoop(ctx, uc, purl, r, preRead); err != nil {
		if aerr := c.abortResumableUpload(ctx, uc, purl); aerr != nil {
			log.Warnf(ctx, "failed to abort resumable upload: %v", aerr)
		}
		return err
	}
	return nil
}

func (c *Client) resumableUploadLoop(ctx context.Context, uc *uploadContext, purl files.PresignedURL, r io.Reader, preRead []byte) error {
	// Buffer one part plus a read-ahead byte: a resumable chunk cannot be empty,
	// so reading one byte past the part tells us whether this is the last chunk
	// (which must be sent with the real total size, not "*").
	minBuffer := uc.partSize + multipartReadAheadBytes
	buffer := slices.Clone(preRead)
	chunkOffset := int64(0)
	retryCount := 0
	noProgress := 0

	for {
		lastChunk := false
		if need := minBuffer - int64(len(buffer)); need > 0 {
			tmp := make([]byte, need)
			n, rerr := io.ReadFull(r, tmp)
			if rerr != nil && rerr != io.EOF && rerr != io.ErrUnexpectedEOF {
				return rerr
			}
			buffer = append(buffer, tmp[:n]...)
			if int64(n) < need {
				lastChunk = true
			}
		}

		var chunkLen int64
		var totalSize string
		if lastChunk {
			chunkLen = int64(len(buffer))
			totalSize = strconv.FormatInt(chunkOffset+chunkLen, 10)
		} else {
			chunkLen = uc.partSize
			totalSize = "*"
		}
		// An empty stream cannot be sent as a resumable upload: a chunk must carry
		// at least one byte, and the range would be the malformed "bytes 0--1/0".
		// Fall back to a single-shot PUT, which creates the empty object. Only
		// reachable at offset 0, since the read-ahead byte folds a non-empty
		// stream's tail into the preceding chunk.
		if chunkLen == 0 {
			return &fallbackToFilesAPI{reason: "empty input stream"}
		}
		chunkLastByte := chunkOffset + chunkLen - 1

		headers := octetStreamHeaders(purl.Headers)
		headers["Content-Range"] = fmt.Sprintf("bytes %d-%d/%s", chunkOffset, chunkLastByte, totalSize)

		if err := uc.limiter.Acquire(ctx); err != nil {
			return err
		}
		resp, rerr := uc.cloud.Attempt(ctx, http.MethodPut, purl.URL, headers, cloudstorage.BytesBody(buffer[:chunkLen]))
		uc.limiter.Release()
		switch {
		case rerr != nil:
			// On a transient transport error, query the server for the confirmed
			// offset and continue from there; otherwise surface the error.
			if retryCount >= multipartMaxRetries || !cloudstorage.IsRetriable(rerr) {
				return rerr
			}
			retryCount++
			status, qerr := c.resumableStatusQuery(ctx, uc, purl)
			if qerr != nil || status == nil {
				return rerr
			}
			resp = status
		case cloudstorage.IsRetryableStatus(resp.StatusCode):
			if retryCount < multipartMaxRetries {
				retryCount++
				if status, qerr := c.resumableStatusQuery(ctx, uc, purl); qerr == nil && status != nil {
					resp = status
				}
			}
		default:
			retryCount = 0
		}

		switch {
		case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated:
			if totalSize == "*" {
				return fmt.Errorf("resumable upload reported complete with status %d before end of stream", resp.StatusCode)
			}
			uc.progress.add(chunkLen)
			return nil
		case resp.StatusCode == http.StatusPermanentRedirect: // 308: chunk accepted, more to come
			confirmed, ok, perr := extractRangeOffset(resp.Header.Get("Range"))
			if perr != nil {
				return perr
			}
			var nextOffset int64
			if ok {
				if confirmed < chunkOffset-1 || confirmed > chunkLastByte {
					return fmt.Errorf("resumable upload confirmed offset %d outside chunk [%d, %d]", confirmed, chunkOffset, chunkLastByte)
				}
				nextOffset = confirmed + 1
			} else {
				if chunkOffset > 0 {
					return fmt.Errorf("resumable upload returned no confirmed offset at chunk offset %d", chunkOffset)
				}
				nextOffset = chunkOffset
			}
			// A 308 confirming no new bytes (nextOffset == chunkOffset) leaves the
			// chunk unsent, and the first switch's default arm resets retryCount, so a
			// server stuck at a fixed offset would loop forever (fs cp sets no context
			// deadline). Bound it with a counter that survives that reset.
			if nextOffset == chunkOffset {
				noProgress++
				if noProgress > multipartMaxRetries {
					return fmt.Errorf("resumable upload made no progress at offset %d after %d attempts", chunkOffset, noProgress)
				}
			} else {
				noProgress = 0
			}
			buffer = buffer[nextOffset-chunkOffset:]
			uc.progress.add(nextOffset - chunkOffset)
			chunkOffset = nextOffset
		case resp.StatusCode == http.StatusPreconditionFailed && (uc.overwrite == nil || !*uc.overwrite):
			return files.ErrAlreadyExists
		default:
			if apiErr := apierr.FromHTTPError(resp.StatusCode, resp.Header, resp.Body); apiErr != nil {
				return apiErr
			}
			return fmt.Errorf("resumable chunk upload failed with status %d", resp.StatusCode)
		}
	}
}

func (c *Client) resumableStatusQuery(ctx context.Context, uc *uploadContext, purl files.PresignedURL) (*cloudstorage.Response, error) {
	return uc.cloud.Attempt(ctx, http.MethodPut, purl.URL, map[string]string{"Content-Range": "bytes */*"}, nil)
}

// extractRangeOffset parses a resumable upload "Range: bytes=0-N" response
// header and returns N. An empty header means the server has confirmed nothing.
func extractRangeOffset(rangeHeader string) (offset int64, ok bool, err error) {
	if rangeHeader == "" {
		return 0, false, nil
	}
	m := resumableRangeRe.FindStringSubmatch(rangeHeader)
	if m == nil {
		return 0, false, fmt.Errorf("cannot parse Range header %q", rangeHeader)
	}
	n, perr := strconv.ParseInt(m[1], 10, 64)
	if perr != nil {
		return 0, false, fmt.Errorf("cannot parse Range header %q: %w", rangeHeader, perr)
	}
	return n, true, nil
}
