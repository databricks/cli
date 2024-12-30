package filer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
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

// Interface to allow mocking of the Databricks API client.
type databricksClient interface {
	Do(ctx context.Context, method, path string, headers map[string]string,
		requestBody any, responseBody any, visitors ...func(*http.Request) error) error
}

// DbfsClient implements the [Filer] interface for the DBFS backend.
type DbfsClient struct {
	workspaceClient *databricks.WorkspaceClient

	apiClient databricksClient

	// File operations will be relative to this path.
	root WorkspaceRootPath
}

func NewDbfsClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return &DbfsClient{
		workspaceClient: w,
		apiClient:       apiClient,

		root: NewWorkspaceRootPath(root),
	}, nil
}

func (w *DbfsClient) putFile(ctx context.Context, path string, overwrite bool, file *os.File) error {
	overwriteField := "False"
	if overwrite {
		overwriteField = "True"
	}

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	err := writer.WriteField("path", path)
	if err != nil {
		return err
	}
	err = writer.WriteField("overwrite", overwriteField)
	if err != nil {
		return err
	}
	contents, err := writer.CreateFormFile("contents", "")
	if err != nil {
		return err
	}

	_, err = io.Copy(contents, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	// Request bodies of Content-Type multipart/form-data must are not supported by
	// the Go SDK directly for DBFS. So we use the Do method directly.
	return w.apiClient.Do(ctx, http.MethodPost, "/api/2.0/dbfs/put", map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}, buf.Bytes(), nil)
}

func (w *DbfsClient) streamFile(ctx context.Context, path string, overwrite bool, reader io.Reader) error {
	fileMode := files.FileModeWrite
	if overwrite {
		fileMode |= files.FileModeOverwrite
	}

	handle, err := w.workspaceClient.Dbfs.Open(ctx, path, fileMode)
	if err != nil {
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return err
		}

		// This API returns a 400 if the file already exists.
		if aerr.StatusCode == http.StatusBadRequest {
			if aerr.ErrorCode == "RESOURCE_ALREADY_EXISTS" {
				return FileAlreadyExistsError{path}
			}
		}

		return err
	}

	_, err = io.Copy(handle, reader)
	cerr := handle.Close()
	if err == nil {
		err = cerr
	}
	return err
}

// TODO CONTINUE:
// 1. Write the unit tests that make sure the filer write method works correctly
//    in either case.
// 2. Write a intergration test that asserts write continues works for big file
//    uploads. Also test the overwrite flag in the integration test.
//    We can change MaxDbfsUploadLimitForPutApi in the test to avoid creating
//    massive test fixtures.

// MaxUploadLimitForPutApi is the maximum size in bytes of a file that can be uploaded
// using the /dbfs/put API. If the file is larger than this limit, the streaming
// API (/dbfs/create and /dbfs/add-block) will be used instead.
var MaxDbfsPutFileSize int64 = 2 * 1024 * 1024

func (w *DbfsClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Issue info call before write because it automatically creates parent directories.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	if !slices.Contains(mode, CreateParentDirectories) {
		_, err = w.workspaceClient.Dbfs.GetStatusByPath(ctx, path.Dir(absPath))
		if err != nil {
			var aerr *apierr.APIError
			if !errors.As(err, &aerr) {
				return err
			}

			// This API returns a 404 if the file doesn't exist.
			if aerr.StatusCode == http.StatusNotFound {
				if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
					return NoSuchDirectoryError{path.Dir(absPath)}
				}
			}

			return err
		}
	}

	localFile, ok := reader.(*os.File)

	// If the source is not a local file, we'll always use the streaming API endpoint.
	if !ok {
		return w.streamFile(ctx, absPath, slices.Contains(mode, OverwriteIfExists), reader)
	}

	stat, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// If the source is a local file, but is too large then we'll use the streaming API endpoint.
	if stat.Size() > MaxDbfsPutFileSize {
		return w.streamFile(ctx, absPath, slices.Contains(mode, OverwriteIfExists), localFile)
	}

	// Use the /dbfs/put API when the file is on the local filesystem
	// and is small enough. This is the most common case when users use the
	// `databricks fs cp` command.
	return w.putFile(ctx, absPath, slices.Contains(mode, OverwriteIfExists), localFile)
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
			return nil, NotAFile{absPath}
		}

		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, FileDoesNotExistError{absPath}
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
		return CannotDeleteRootError{}
	}

	// Issue info call before delete because delete succeeds if the specified path doesn't exist.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	_, err = w.workspaceClient.Dbfs.GetStatusByPath(ctx, absPath)
	if err != nil {
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return FileDoesNotExistError{absPath}
			}
		}

		return err
	}

	recursive := false
	if slices.Contains(mode, DeleteRecursively) {
		recursive = true
	}

	err = w.workspaceClient.Dbfs.Delete(ctx, files.Delete{
		Path:      absPath,
		Recursive: recursive,
	})

	// Return early on success.
	if err == nil {
		return nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return err
	}

	switch aerr.StatusCode {
	case http.StatusBadRequest:
		// Anecdotally, this error is returned when attempting to delete a non-empty directory.
		if aerr.ErrorCode == "IO_ERROR" {
			return DirectoryNotEmptyError{absPath}
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
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, NoSuchDirectoryError{absPath}
			}
		}

		return nil, err
	}

	if len(res.Files) == 1 && res.Files[0].Path == absPath {
		return nil, NotADirectory{absPath}
	}

	info := make([]fs.DirEntry, len(res.Files))
	for i, v := range res.Files {
		info[i] = dbfsDirEntry{dbfsFileInfo: dbfsFileInfo{fi: v}}
	}

	// Sort by name for parity with os.ReadDir.
	sort.Slice(info, func(i, j int) bool { return info[i].Name() < info[j].Name() })
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
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, FileDoesNotExistError{absPath}
			}
		}

		return nil, err
	}

	return dbfsFileInfo{*info}, nil
}
