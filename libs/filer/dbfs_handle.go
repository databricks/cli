package filer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/databricks/databricks-sdk-go/service/dbfs"
)

// DbfsFileMode conveys user intent when opening a file.
type DbfsFileMode int

const (
	// Exactly one of DbfsRead or DbfsWrite must be specified.
	DbfsRead DbfsFileMode = 1 << iota
	DbfsWrite
	DbfsOverwrite
)

// Maximum read or write length for the DBFS API.
const maxDbfsBlockSize = 1024 * 1024

type dbfsReader struct {
	size   int64
	offset int64
}

type dbfsWriter struct {
	handle int64
}

type dbfsHandle struct {
	ctx  context.Context
	api  *dbfs.DbfsAPI
	path string

	*dbfsReader
	*dbfsWriter
}

// Implement the [io.Reader] interface.
func (h *dbfsHandle) Read(p []byte) (int, error) {
	r := h.dbfsReader
	if r == nil {
		return 0, fmt.Errorf("dbfs file not open for reading")
	}

	if r.offset >= r.size {
		return 0, io.EOF
	}

	res, err := h.api.Read(h.ctx, dbfs.Read{
		Path:   h.path,
		Length: len(p),
		Offset: int(r.offset), // TODO: make int32/in64 work properly
	})
	if err != nil {
		return 0, fmt.Errorf("dbfs read: %w", err)
	}

	// The guard against offset >= size happens above, so this can only happen
	// if the file is modified or truncated while reading. If this happens,
	// the read contents will likely be corrupted, so we return an error.
	if res.BytesRead == 0 {
		return 0, fmt.Errorf("dbfs read: unexpected EOF at offset %d (size %d)", r.offset, r.size)
	}

	r.offset += res.BytesRead
	return base64.StdEncoding.Decode(p, []byte(res.Data))
}

// Implement the [io.WriterTo] interface.
func (h *dbfsHandle) WriteTo(w io.Writer) (int64, error) {
	r := h.dbfsReader
	if r == nil {
		return 0, fmt.Errorf("dbfs file not open for reading")
	}

	buf := make([]byte, maxDbfsBlockSize)
	ntotal := int64(0)
	for {
		nread, err := h.Read(buf)
		if err != nil {
			// EOF on read means we're done.
			// For writers being done means returning a nil error.
			if err == io.EOF {
				err = nil
			}
			return ntotal, err
		}
		nwritten, err := io.Copy(w, bytes.NewReader(buf[:nread]))
		ntotal += nwritten
		if err != nil {
			return ntotal, err
		}
	}
}

// Implement the [io.Writer] interface.
func (h *dbfsHandle) Write(p []byte) (int, error) {
	w := h.dbfsWriter
	if w == nil {
		return 0, fmt.Errorf("dbfs: file not open for writing")
	}

	err := h.api.AddBlock(h.ctx, dbfs.AddBlock{
		Data:   base64.StdEncoding.EncodeToString(p),
		Handle: w.handle,
	})
	if err != nil {
		return 0, fmt.Errorf("dbfs: add block: %w", err)
	}
	return len(p), nil
}

// Implement the [io.ReaderFrom] interface.
func (h *dbfsHandle) ReadFrom(r io.Reader) (int64, error) {
	w := h.dbfsWriter
	if w == nil {
		return 0, fmt.Errorf("dbfs: file not open for writing")
	}

	buf := make([]byte, maxDbfsBlockSize)
	ntotal := int64(0)
	for {
		nread, err := r.Read(buf)
		if err != nil {
			// EOF on read means we're done.
			// For writers being done means returning a nil error.
			if err == io.EOF {
				err = nil
			}
			return ntotal, err
		}

		nwritten, err := h.Write(buf[:nread])
		ntotal += int64(nwritten)
		if err != nil {
			return ntotal, err
		}
	}
}

// Implement the [io.Closer] interface.
func (h *dbfsHandle) Close() error {
	w := h.dbfsWriter
	if w == nil {
		return fmt.Errorf("dbfs: file not open for writing")
	}

	err := h.api.CloseByHandle(h.ctx, w.handle)
	if err != nil {
		return fmt.Errorf("dbfs: close: %w", err)
	}

	return nil
}

func (h *dbfsHandle) openForRead(mode DbfsFileMode) (*dbfsHandle, error) {
	res, err := h.api.GetStatusByPath(h.ctx, h.path)
	if err != nil {
		return nil, err
	}
	h.dbfsReader = &dbfsReader{
		size: res.FileSize,
	}
	return h, nil
}

func (h *dbfsHandle) openForWrite(mode DbfsFileMode) (*dbfsHandle, error) {
	res, err := h.api.Create(h.ctx, dbfs.Create{
		Path:      h.path,
		Overwrite: (mode & DbfsOverwrite) != 0,
	})
	if err != nil {
		return nil, err
	}
	h.dbfsWriter = &dbfsWriter{
		handle: res.Handle,
	}
	return h, nil
}

// OpenFile opens a remote DBFS file for reading or writing.
// The returned object implements relevant [io] interfaces for convenient
// integration with other code that reads or writes bytes.
//
// The [io.WriterTo] interface is provided and maximizes throughput for
// bulk reads by reading data with the DBFS maximum read chunk size of 1MB.
// Similarly, the [io.ReaderFrom] interface is provided for bulk writing.
//
// A file opened for writing must always be closed.
func OpenFile(ctx context.Context, api *dbfs.DbfsAPI, path string, mode DbfsFileMode) (*dbfsHandle, error) {
	h := &dbfsHandle{
		ctx:  ctx,
		api:  api,
		path: path,
	}

	isRead := (mode & DbfsRead) != 0
	isWrite := (mode & DbfsWrite) != 0
	if isRead && isWrite {
		return nil, fmt.Errorf("dbfs: cannot open file for reading and writing at the same time")
	}
	if isRead {
		return h.openForRead(mode)
	}
	if isWrite {
		return h.openForWrite(mode)
	}

	// No mode specifed. The caller should be explicit so we return an error.
	return nil, fmt.Errorf("dbfs: must specify DbfsRead or DbfsWrite")
}
