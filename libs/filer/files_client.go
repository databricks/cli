package filer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/files"
)

// Type that implements fs.FileInfo for the Files API.
// This is required for the filer.Stat() method.
type filesApiFileInfo struct {
	absPath      string
	isDir        bool
	fileSize     int64
	lastModified int64
}

func (info filesApiFileInfo) Name() string {
	return path.Base(info.absPath)
}

func (info filesApiFileInfo) Size() int64 {
	return info.fileSize
}

func (info filesApiFileInfo) Mode() fs.FileMode {
	mode := fs.ModePerm
	if info.isDir {
		mode |= fs.ModeDir
	}
	return mode
}

func (info filesApiFileInfo) ModTime() time.Time {
	return time.UnixMilli(info.lastModified)
}

func (info filesApiFileInfo) IsDir() bool {
	return info.isDir
}

func (info filesApiFileInfo) Sys() any {
	return nil
}

// Type that implements fs.DirEntry for the Files API.
// This is required for the filer.ReadDir() method.
type filesApiDirEntry struct {
	i filesApiFileInfo
}

func (e filesApiDirEntry) Name() string {
	return e.i.Name()
}

func (e filesApiDirEntry) IsDir() bool {
	return e.i.IsDir()
}

func (e filesApiDirEntry) Type() fs.FileMode {
	return e.i.Mode()
}

func (e filesApiDirEntry) Info() (fs.FileInfo, error) {
	return e.i, nil
}

// FilesClient implements the [Filer] interface for the Files API backend.
type FilesClient struct {
	workspaceClient *databricks.WorkspaceClient
	apiClient       *client.DatabricksClient

	// File operations will be relative to this path.
	root WorkspaceRootPath
}

func NewFilesClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}

	return &FilesClient{
		workspaceClient: w,
		apiClient:       apiClient,

		root: NewWorkspaceRootPath(root),
	}, nil
}

func (w *FilesClient) urlPath(name string) (string, string, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return "", "", err
	}

	// The user specified part of the path must be escaped.
	urlPath := fmt.Sprintf(
		"/api/2.0/fs/files/%s",
		url.PathEscape(strings.TrimLeft(absPath, "/")),
	)

	return absPath, urlPath, nil
}

func (w *FilesClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, urlPath, err := w.urlPath(name)
	if err != nil {
		return err
	}

	overwrite := slices.Contains(mode, OverwriteIfExists)
	urlPath = fmt.Sprintf("%s?overwrite=%t", urlPath, overwrite)
	headers := map[string]string{"Content-Type": "application/octet-stream"}
	err = w.apiClient.Do(ctx, http.MethodPut, urlPath, headers, reader, nil)

	// Return early on success.
	if err == nil {
		return nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return err
	}

	// This API returns 409 if the file already exists, when the object type is file
	if aerr.StatusCode == http.StatusConflict {
		return FileAlreadyExistsError{absPath}
	}

	return err
}

func (w *FilesClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, urlPath, err := w.urlPath(name)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = w.apiClient.Do(ctx, http.MethodGet, urlPath, nil, nil, &buf)

	// Return early on success.
	if err == nil {
		return io.NopCloser(&buf), nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return nil, err
	}

	// This API returns a 404 if the specified path does not exist.
	if aerr.StatusCode == http.StatusNotFound {
		return nil, FileDoesNotExistError{absPath}
	}

	// This API returns a 409 if the underlying path is a directory.
	if aerr.StatusCode == http.StatusConflict {
		return nil, NotAFile{absPath}
	}

	return nil, err
}

func (w *FilesClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	if slices.Contains(mode, DeleteRecursively) {
		return fmt.Errorf("files API does not support recursive delete")
	}

	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return CannotDeleteRootError{}
	}

	info, err := w.Stat(ctx, name)
	if err != nil {
		return err
	}

	if info.IsDir() {
		err = w.workspaceClient.Files.DeleteDirectoryByDirectoryPath(ctx, absPath)

		// This API returns a 400 if the directory is not empty
		var aerr *apierr.APIError
		if errors.As(err, &aerr) && aerr.StatusCode == http.StatusBadRequest {
			return DirectoryNotEmptyError{absPath}
		}
		return err
	}

	err = w.workspaceClient.Files.DeleteByFilePath(ctx, absPath)

	// This API returns a 404 if the specified path does not exist.
	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
		return FileDoesNotExistError{absPath}
	}
	return err
}

func (w *FilesClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	iter := w.workspaceClient.Files.ListDirectoryContents(ctx, files.ListDirectoryContentsRequest{
		DirectoryPath: absPath,
	})

	files, err := listing.ToSlice(ctx, iter)
	var apierr *apierr.APIError

	// This API returns a 404 if the specified path does not exist.
	if errors.As(err, &apierr) && apierr.StatusCode == http.StatusNotFound {
		return nil, NoSuchDirectoryError{absPath}
	}
	// This API returns 409 if the underlying path is a file.
	if errors.As(err, &apierr) && apierr.StatusCode == http.StatusConflict {
		return nil, NotADirectory{absPath}
	}
	if err != nil {
		return nil, err
	}

	entries := make([]fs.DirEntry, len(files))
	for i, file := range files {
		entries[i] = filesApiDirEntry{
			i: filesApiFileInfo{
				absPath:      file.Path,
				isDir:        file.IsDirectory,
				fileSize:     file.FileSize,
				lastModified: file.LastModified,
			},
		}
	}

	// Sort by name for parity with os.ReadDir.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (w *FilesClient) Mkdir(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	err = w.workspaceClient.Files.CreateDirectory(ctx, files.CreateDirectoryRequest{
		DirectoryPath: absPath,
	})

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusConflict {
		return FileAlreadyExistsError{absPath}
	}

	return err
}

func (w *FilesClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, urlPath, err := w.urlPath(name)
	if err != nil {
		return nil, err
	}

	err = w.apiClient.Do(ctx, http.MethodHead, urlPath, nil, nil, nil)

	// If the HEAD requests succeeds, the file exists.
	if err == nil {
		return filesApiFileInfo{absPath: absPath, isDir: false}, nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return nil, err
	}

	// This API returns a 404 if the specified path does not exist.
	if aerr.StatusCode == http.StatusNotFound {
		return nil, FileDoesNotExistError{absPath}
	}

	// This API returns 409 if the underlying path is a directory.
	if aerr.StatusCode == http.StatusConflict {
		return filesApiFileInfo{absPath: absPath, isDir: true}, nil
	}

	return nil, err
}
