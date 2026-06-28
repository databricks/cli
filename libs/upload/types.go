package upload

// Shared buffering and cloud-leg header helpers. The Files API wire types and
// the control-plane header conversion live in the files package; the
// cloud-transport pieces (part bodies, retry classification, presigned-URL-expiry
// detection) live in the cloudstorage package.

import (
	"io"
	"maps"
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
