package filer

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/databricks/cli/libs/utils"
	files "github.com/databricks/sdk-go/files/v2"
)

// dbfsChunkSize is the largest block the DBFS add-block and read APIs accept.
const dbfsChunkSize = 1 << 20

var errDbfsOpenDirectory = errors.New("cannot open directory for reading")

// dbfsWriter streams a DBFS file through the create / add-block / close
// handle API, which caps each block at 1 MB.
type dbfsWriter struct {
	ctx    context.Context
	client *files.Client
	handle int64
	buf    bytes.Buffer
}

func openDbfsWriter(ctx context.Context, client *files.Client, path string, overwrite bool) (*dbfsWriter, error) {
	resp, err := client.Create(ctx, files.CreateRequest{Path: new(path), Overwrite: new(overwrite)})
	if err != nil {
		return nil, err
	}
	return &dbfsWriter{ctx: ctx, client: client, handle: utils.Deref(resp.Handle)}, nil
}

func (w *dbfsWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for w.buf.Len() >= dbfsChunkSize {
		if err := w.flush(dbfsChunkSize); err != nil {
			return 0, err
		}
	}
	return len(p), nil
}

func (w *dbfsWriter) flush(n int) error {
	_, err := w.client.AddBlock(w.ctx, files.AddBlockRequest{Handle: new(w.handle), Data: w.buf.Next(n)})
	return err
}

func (w *dbfsWriter) Close() error {
	if w.buf.Len() > 0 {
		if err := w.flush(w.buf.Len()); err != nil {
			return err
		}
	}
	_, err := w.client.Close(w.ctx, files.CloseRequest{Handle: new(w.handle)})
	return err
}

// dbfsReader reads a DBFS file in 1 MB chunks through the read API.
type dbfsReader struct {
	ctx    context.Context
	client *files.Client
	path   string
	offset int64
	buf    []byte
	eof    bool
}

func openDbfsReader(ctx context.Context, client *files.Client, path string) (*dbfsReader, error) {
	info, err := client.GetStatus(ctx, files.GetStatusRequest{Path: new(path)})
	if err != nil {
		return nil, err
	}
	if utils.Deref(info.IsDir) {
		return nil, errDbfsOpenDirectory
	}
	return &dbfsReader{ctx: ctx, client: client, path: path}, nil
}

func (r *dbfsReader) Read(p []byte) (int, error) {
	if len(r.buf) == 0 {
		if r.eof {
			return 0, io.EOF
		}
		resp, err := r.client.Read(r.ctx, files.ReadRequest{
			Path:   new(r.path),
			Offset: new(r.offset),
			Length: new(int64(dbfsChunkSize)),
		})
		if err != nil {
			return 0, err
		}
		n := utils.Deref(resp.BytesRead)
		if n == 0 {
			r.eof = true
			return 0, io.EOF
		}
		r.offset += n
		r.buf = resp.Data
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}
