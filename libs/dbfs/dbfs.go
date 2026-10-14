// Package dbfs provides streaming file handles over the DBFS API, built on the
// sdk-go files client. It ports the handle helpers of the previous SDK
// (DbfsAPI.Open, ReadFile, WriteFile, RecursiveList), which sdk-go lacks.
package dbfs

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/utils"
	files "github.com/databricks/sdk-go/files/v2"
)

// FileMode conveys user intent when opening a file.
type FileMode int

const (
	// Exactly one of FileModeRead or FileModeWrite must be specified.
	FileModeRead FileMode = 1 << iota
	FileModeWrite
	FileModeOverwrite
)

// Maximum read or write length for the DBFS API.
const maxBlockSize = 1024 * 1024

// Handle is returned by [Open].
type Handle interface {
	io.ReadWriteCloser
	io.WriterTo
	io.ReaderFrom
}

type reader struct {
	size   int64
	offset int64
}

type writer struct {
	handle int64
}

type handle struct {
	ctx    context.Context
	api    *files.Client
	path   string
	reader *reader
	writer *writer
}

func (h *handle) errorf(format string, a ...any) error {
	return fmt.Errorf("dbfs %s: %s", h.path, fmt.Sprintf(format, a...))
}

func (h *handle) wrap(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("dbfs %s: %w", h.path, err)
}

func (h *handle) Read(p []byte) (int, error) {
	r := h.reader
	if r == nil {
		return 0, fmt.Errorf("dbfs: file not open for reading")
	}
	var ntotal int
	for ntotal < len(p) {
		if r.offset >= r.size {
			return ntotal, io.EOF
		}
		chunk := p[ntotal:]
		if len(chunk) > maxBlockSize {
			chunk = chunk[:maxBlockSize]
		}
		res, err := h.api.Read(h.ctx, files.ReadRequest{
			Path:   new(h.path),
			Length: new(int64(len(chunk))),
			Offset: new(r.offset),
		})
		if err != nil {
			return ntotal, h.wrap(err)
		}
		// Reading past the size checked above means the file was modified or
		// truncated concurrently; the contents would be corrupted.
		if utils.Deref(res.BytesRead) == 0 {
			return ntotal, h.errorf("unexpected EOF at offset %d (size %d)", r.offset, r.size)
		}
		n := copy(chunk, res.Data)
		ntotal += n
		r.offset += int64(n)
	}
	return ntotal, nil
}

func (h *handle) Write(p []byte) (int, error) {
	w := h.writer
	if w == nil {
		return 0, fmt.Errorf("dbfs: file not open for writing")
	}
	var ntotal int
	for ntotal < len(p) {
		chunk := p[ntotal:]
		if len(chunk) > maxBlockSize {
			chunk = chunk[:maxBlockSize]
		}
		_, err := h.api.AddBlock(h.ctx, files.AddBlockRequest{
			Handle: new(w.handle),
			Data:   chunk,
		})
		if err != nil {
			return ntotal, h.wrap(err)
		}
		ntotal += len(chunk)
	}
	return ntotal, nil
}

func (h *handle) Close() error {
	w := h.writer
	if w == nil {
		return fmt.Errorf("dbfs: file not open for writing")
	}
	_, err := h.api.Close(h.ctx, files.CloseRequest{Handle: new(w.handle)})
	return h.wrap(err)
}

// WriteTo maximizes throughput by reading in chunks of the DBFS maximum size.
func (h *handle) WriteTo(w io.Writer) (int64, error) {
	if h.reader == nil {
		return 0, fmt.Errorf("dbfs: file not open for reading")
	}
	// Plain io.Reader/io.Writer wrappers avoid recursing into WriteTo/ReadFrom.
	ior := struct{ io.Reader }{h}
	iow := struct{ io.Writer }{w}
	return bufio.NewReaderSize(ior, maxBlockSize).WriteTo(iow)
}

// ReadFrom maximizes throughput by writing in chunks of the DBFS maximum size.
func (h *handle) ReadFrom(r io.Reader) (int64, error) {
	if h.writer == nil {
		return 0, fmt.Errorf("dbfs: file not open for writing")
	}
	ior := struct{ io.Reader }{r}
	iow := struct{ io.Writer }{h}
	bw := bufio.NewWriterSize(iow, maxBlockSize)
	n, err := bw.ReadFrom(ior)
	if err != nil {
		return n, err
	}
	return n, bw.Flush()
}

// Open opens a remote DBFS file for reading or writing. A file opened for
// writing must always be closed.
func Open(ctx context.Context, api *files.Client, path string, mode FileMode) (Handle, error) {
	h := &handle{ctx: ctx, api: api, path: path}
	isRead := mode&FileModeRead != 0
	isWrite := mode&FileModeWrite != 0
	if isRead == isWrite {
		return nil, fmt.Errorf("dbfs open: must specify dbfs.FileModeRead or dbfs.FileModeWrite")
	}
	if isRead {
		res, err := api.GetStatus(ctx, files.GetStatusRequest{Path: new(path)})
		if err != nil {
			return nil, fmt.Errorf("dbfs open: %w", err)
		}
		if utils.Deref(res.IsDir) {
			return nil, fmt.Errorf("dbfs open: cannot open directory for reading")
		}
		h.reader = &reader{size: utils.Deref(res.FileSize)}
	} else {
		res, err := api.Create(ctx, files.CreateRequest{
			Path:      new(path),
			Overwrite: new(mode&FileModeOverwrite != 0),
		})
		if err != nil {
			return nil, fmt.Errorf("dbfs open: %w", err)
		}
		h.writer = &writer{handle: utils.Deref(res.Handle)}
	}
	return h, nil
}

// ReadFile is identical to [os.ReadFile] but for DBFS.
func ReadFile(ctx context.Context, api *files.Client, name string) ([]byte, error) {
	h, err := Open(ctx, api, name, FileModeRead)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, h.(*handle).reader.size)
	_, err = h.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf, nil
}

// WriteFile is identical to [os.WriteFile] but for DBFS.
func WriteFile(ctx context.Context, api *files.Client, name string, data []byte) error {
	h, err := Open(ctx, api, name, FileModeWrite|FileModeOverwrite)
	if err != nil {
		return err
	}
	_, err = h.Write(data)
	cerr := h.Close()
	if err == nil {
		err = cerr
	}
	return err
}

// RecursiveList traverses the DBFS tree and returns all non-directory objects
// under the path.
func RecursiveList(ctx context.Context, api *files.Client, path string) ([]files.FileInfo, error) {
	var results []files.FileInfo
	queue := []string{path}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		res, err := api.List(ctx, files.ListStatusRequest{Path: new(path)})
		if apierrs.IsMissing(err) {
			// skip on path deleted during iteration
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", path, err)
		}
		for _, v := range res.Files {
			if utils.Deref(v.IsDir) {
				queue = append(queue, utils.Deref(v.Path))
				continue
			}
			results = append(results, v)
		}
	}
	return results, nil
}
