package cloudstorage

import (
	"bytes"
	"io"
)

// Body supplies a request's bytes and can re-supply them for each retry attempt,
// so a request is retried without the caller buffering it again.
type Body interface {
	// Reader returns a reader over the body positioned at the start. It is called
	// once per attempt.
	Reader() (io.Reader, error)
	// Size reports the body length in bytes, for Content-Length and the
	// short-read guard.
	Size() int64
}

// BytesBody returns a Body backed by an in-memory buffer.
func BytesBody(data []byte) Body {
	return bytesBody{data}
}

type bytesBody struct {
	data []byte
}

func (b bytesBody) Reader() (io.Reader, error) {
	return bytes.NewReader(b.data), nil
}

func (b bytesBody) Size() int64 {
	return int64(len(b.data))
}

// SectionBody returns a Body that reads length bytes starting at offset from ra
// on each attempt. ra may be shared across concurrent bodies: positioned reads
// via io.ReaderAt are safe to run in parallel.
func SectionBody(ra io.ReaderAt, offset, length int64) Body {
	return sectionBody{ra: ra, offset: offset, length: length}
}

type sectionBody struct {
	ra     io.ReaderAt
	offset int64
	length int64
}

func (s sectionBody) Reader() (io.Reader, error) {
	return io.NewSectionReader(s.ra, s.offset, s.length), nil
}

func (s sectionBody) Size() int64 {
	return s.length
}
