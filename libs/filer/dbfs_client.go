package filer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/dbfs"
)

var b64 = base64.StdEncoding

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
	api  dbfs.DbfsAPI
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
	return b64.Decode(p, []byte(res.Data))
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
		return 0, fmt.Errorf("dbfs file not open for writing")
	}

	err := h.api.AddBlock(h.ctx, dbfs.AddBlock{
		Data:   b64.EncodeToString(p),
		Handle: w.handle,
	})
	if err != nil {
		return 0, fmt.Errorf("dbfs add block: %w", err)
	}
	return len(p), nil
}

// Implement the [io.ReaderFrom] interface.
func (h *dbfsHandle) ReadFrom(r io.Reader) (int64, error) {
	w := h.dbfsWriter
	if w == nil {
		return 0, fmt.Errorf("dbfs file not open for writing")
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
		return fmt.Errorf("dbfs file not open for writing")
	}

	err := h.api.CloseByHandle(h.ctx, w.handle)
	if err != nil {
		return fmt.Errorf("dbfs close: %w", err)
	}

	return nil
}

func newDbfsHandleForReading(ctx context.Context, w *databricks.WorkspaceClient, path string) (io.Reader, error) {
	info, err := w.Dbfs.GetStatusByPath(ctx, path)
	if err != nil {
		return nil, err
	}

	return &dbfsHandle{
		ctx:  ctx,
		api:  *w.Dbfs,
		path: path,

		dbfsReader: &dbfsReader{
			size: info.FileSize,
		},
	}, nil
}

func newDbfsHandleForWriting(ctx context.Context, w *databricks.WorkspaceClient, path string) (io.WriteCloser, error) {
	res, err := w.Dbfs.Create(ctx, dbfs.Create{
		Path:      path,
		Overwrite: false,
	})
	if err != nil {
		return nil, err
	}

	return &dbfsHandle{
		ctx:  ctx,
		api:  *w.Dbfs,
		path: path,

		dbfsWriter: &dbfsWriter{
			handle: res.Handle,
		},
	}, nil
}

// DbfsClient implements a
type DbfsClient struct {
	workspaceClient *databricks.WorkspaceClient

	// File operations will be relative to this path.
	root RootPath
}

func NewDbfsClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	return &DbfsClient{
		workspaceClient: w,

		root: NewRootPath(root),
	}, nil
}

func (w *DbfsClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	dbfsHandle, err := newDbfsHandleForWriting(ctx, w.workspaceClient, absPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(dbfsHandle, reader)
	cerr := dbfsHandle.Close()
	if err == nil {
		err = cerr
	}
	return err
}

func (w *DbfsClient) Read(ctx context.Context, name string) (io.Reader, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	return newDbfsHandleForReading(ctx, w.workspaceClient, absPath)
}

func (w *DbfsClient) Delete(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	return w.workspaceClient.Dbfs.Delete(ctx, dbfs.Delete{
		Path:      absPath,
		Recursive: false,
	})
}
