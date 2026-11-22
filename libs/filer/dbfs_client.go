package filer

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/files"
)

// Type that implements fs.DirEntry for DBFS.
type dbfsDirEntry struct {
	dbfsFileInfo
}

func (entry dbfsDirEntry) Type() fs.FileMode {
	return entry.Mode()
}

func (entry dbfsDirEntry) Info() (fs.FileInfo, error) {
	return entry.dbfsFileInfo, nil
}

// Type that implements fs.FileInfo for DBFS.
type dbfsFileInfo struct {
	fi files.FileInfo
}

func (info dbfsFileInfo) Name() string {
	return path.Base(info.fi.Path)
}

func (info dbfsFileInfo) Size() int64 {
	return info.fi.FileSize
}

func (info dbfsFileInfo) Mode() fs.FileMode {
	mode := fs.ModePerm
	if info.fi.IsDir {
		mode |= fs.ModeDir
	}
	return mode
}

func (info dbfsFileInfo) ModTime() time.Time {
	return time.UnixMilli(info.fi.ModificationTime)
}

func (info dbfsFileInfo) IsDir() bool {
	return info.fi.IsDir
}

func (info dbfsFileInfo) Sys() any {
	return info.fi
}

// DbfsClient implements the [Filer] interface for the DBFS backend.
type DbfsClient struct {
	workspaceClient *databricks.WorkspaceClient

	// File operations will be relative to this path.
	root WorkspaceRootPath
}

func NewDbfsClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	return &DbfsClient{
		workspaceClient: w,

		root: NewWorkspaceRootPath(root),
	}, nil
}

func (w *DbfsClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	fileMode := files.FileModeWrite
	if slices.Contains(mode, OverwriteIfExists) {
		fileMode |= files.FileModeOverwrite
	}

	// Issue info call before write because it automatically creates parent directories.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	if !slices.Contains(mode, CreateParentDirectories) {
		_, err = w.workspaceClient.Dbfs.GetStatusByPath(ctx, path.Dir(absPath))
		if err != nil {
			aerr, ok := errors.AsType[*apierr.APIError](err)
			if !ok {
				return err
			}

			// This API returns a 404 if the file doesn't exist.
			if aerr.StatusCode == http.StatusNotFound {
				if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
					return noSuchDirectoryError{path.Dir(absPath)}
				}
			}

			return err
		}
	}

	src := newDbfsUploadSource(reader)
	err = w.upload(ctx, absPath, fileMode, src)
	if !isDbfsStreamLost(err) {
		return err
	}

	// The backend forgot the stream handle mid-upload, so the file is incomplete.
	// Restart the whole create/add-block/close sequence once. The first create
	// succeeded, so whatever is at the path now is this upload's own partial file;
	// overwrite it even if the caller did not ask to overwrite an existing file.
	// A file another writer created at the same path in between is replaced too.
	if !src.rewind() {
		return err
	}
	log.Debugf(ctx, "dbfs write to %s lost its stream handle, restarting the upload: %s", absPath, err)
	return w.upload(ctx, absPath, fileMode|files.FileModeOverwrite, src)
}

// upload writes the content of src to absPath with one DBFS stream (create, add-block, close).
func (w *DbfsClient) upload(ctx context.Context, absPath string, fileMode files.FileMode, src io.Reader) error {
	handle, err := w.workspaceClient.Dbfs.Open(ctx, absPath, fileMode)
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return err
		}

		// This API returns a 400 if the file already exists.
		if aerr.StatusCode == http.StatusBadRequest {
			if aerr.ErrorCode == "RESOURCE_ALREADY_EXISTS" {
				return fileAlreadyExistsError{absPath}
			}
		}

		return err
	}

	_, err = io.Copy(handle, src)
	cerr := handle.Close()
	if err == nil {
		err = cerr
	}

	return err
}

// isDbfsStreamLost reports whether err says that the DBFS backend no longer knows
// the stream handle of an upload. It happens transiently in CI on every cloud,
// for files of a few bytes, a few seconds after the handle was created.
func isDbfsStreamLost(err error) bool {
	aerr, ok := errors.AsType[*apierr.APIError](err)
	return ok && aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" && strings.HasPrefix(aerr.Message, "No such stream with handle")
}

// dbfsReplayBufferSize is the largest non-seekable source that a DBFS upload
// keeps in memory so it can restart after losing its stream handle. It caps
// memory, it is not an API limit. An upload that has read more than this from a
// non-seekable source is not restarted.
// A variable so tests can lower it.
var dbfsReplayBufferSize = 8 * 1024 * 1024 // 8 MiB

// dbfsUploadSource reads an upload's source and can rewind it to the start once.
// A seekable source is rewound with Seek. A non-seekable one (a remote file
// download, or a pipe, which is an *os.File whose Seek fails) is copied into a
// buffer while it is read, up to dbfsReplayBufferSize bytes.
type dbfsUploadSource struct {
	r io.Reader

	seeker io.Seeker
	start  int64

	buf      bytes.Buffer
	overflow bool
	replay   io.Reader
}

func newDbfsUploadSource(r io.Reader) *dbfsUploadSource {
	src := &dbfsUploadSource{r: r}
	if s, ok := r.(io.Seeker); ok {
		if start, err := s.Seek(0, io.SeekCurrent); err == nil {
			src.seeker = s
			src.start = start
		}
	}
	return src
}

func (s *dbfsUploadSource) Read(p []byte) (int, error) {
	if s.replay != nil {
		return s.replay.Read(p)
	}
	n, err := s.r.Read(p)
	if s.seeker == nil && !s.overflow {
		if s.buf.Len()+n > dbfsReplayBufferSize {
			s.overflow = true
			s.buf = bytes.Buffer{}
		} else {
			s.buf.Write(p[:n])
		}
	}
	return n, err
}

// rewind makes the next Read start from the beginning of the source again.
// It returns false if that is not possible. It works only once.
func (s *dbfsUploadSource) rewind() bool {
	if s.replay != nil {
		return false
	}
	if s.seeker != nil {
		if _, err := s.seeker.Seek(s.start, io.SeekStart); err != nil {
			return false
		}
		s.replay = s.r
		return true
	}
	if s.overflow {
		return false
	}
	s.replay = io.MultiReader(bytes.NewReader(s.buf.Bytes()), s.r)
	return true
}

func (w *DbfsClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	handle, err := w.workspaceClient.Dbfs.Open(ctx, absPath, files.FileModeRead)
	if err != nil {
		// Return error if file is a directory
		if strings.Contains(err.Error(), "cannot open directory for reading") {
			return nil, notAFile{absPath}
		}

		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, fileDoesNotExistError{absPath}
			}
		}

		return nil, err
	}

	// A DBFS handle open for reading does not need to be closed.
	return io.NopCloser(handle), nil
}

func (w *DbfsClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return cannotDeleteRootError{}
	}

	// Issue info call before delete because delete succeeds if the specified path doesn't exist.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	_, err = w.workspaceClient.Dbfs.GetStatusByPath(ctx, absPath)
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return fileDoesNotExistError{absPath}
			}
		}

		return err
	}

	recursive := slices.Contains(mode, DeleteRecursively)

	err = w.workspaceClient.Dbfs.Delete(ctx, files.Delete{
		Path:      absPath,
		Recursive: recursive,
	})

	// Return early on success.
	if err == nil {
		return nil
	}

	// Special handling of this error only if it is an API error.
	aerr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return err
	}

	switch aerr.StatusCode {
	case http.StatusBadRequest:
		// Anecdotally, this error is returned when attempting to delete a non-empty directory.
		if aerr.ErrorCode == "IO_ERROR" {
			return directoryNotEmptyError{absPath}
		}

		// Since 17th december we are observing the backend return an error_code of BAD_REQUEST
		// instead of IO_ERROR. Doing direct comparison on the error string instead.
		if strings.Contains(aerr.Message, "Directory is not empty. Use recursive delete or remove contents first.") {
			return directoryNotEmptyError{absPath}
		}
	}

	return err
}

func (w *DbfsClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	res, err := w.workspaceClient.Dbfs.ListByPath(ctx, absPath)
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, noSuchDirectoryError{absPath}
			}
		}

		return nil, err
	}

	if len(res.Files) == 1 && res.Files[0].Path == absPath {
		return nil, notADirectory{absPath}
	}

	info := make([]fs.DirEntry, len(res.Files))
	for i, v := range res.Files {
		info[i] = dbfsDirEntry{fi: v}
	}

	// Sort by name for parity with os.ReadDir.
	slices.SortFunc(info, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return info, nil
}

func (w *DbfsClient) Mkdir(ctx context.Context, name string) error {
	dirPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	return w.workspaceClient.Dbfs.MkdirsByPath(ctx, dirPath)
}

func (w *DbfsClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	info, err := w.workspaceClient.Dbfs.GetStatusByPath(ctx, absPath)
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, fileDoesNotExistError{absPath}
			}
		}

		return nil, err
	}

	return dbfsFileInfo{*info}, nil
}
