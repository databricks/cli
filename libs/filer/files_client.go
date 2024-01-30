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
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
)

// Type that implements fs.FileInfo for the Files API.
type filesApiFileInfo struct {
	absPath string
	isDir   bool
}

func (info filesApiFileInfo) Name() string {
	return path.Base(info.absPath)
}

func (info filesApiFileInfo) Size() int64 {
	// No way to get the file size in the Files API.
	return 0
}

func (info filesApiFileInfo) Mode() fs.FileMode {
	mode := fs.ModePerm
	if info.isDir {
		mode |= fs.ModeDir
	}
	return mode
}

func (info filesApiFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (info filesApiFileInfo) IsDir() bool {
	return info.isDir
}

func (info filesApiFileInfo) Sys() any {
	return nil
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

	return nil, err
}

func (w *FilesClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, urlPath, err := w.urlPath(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return CannotDeleteRootError{}
	}

	err = w.apiClient.Do(ctx, http.MethodDelete, urlPath, nil, nil, nil)

	// Return early on success.
	if err == nil {
		return nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return err
	}

	// This API returns a 404 if the specified path does not exist.
	if aerr.StatusCode == http.StatusNotFound {
		return FileDoesNotExistError{absPath}
	}

	// This API returns 409 if the underlying path is a directory.
	if aerr.StatusCode == http.StatusConflict {
		return DirectoryNotEmptyError{absPath}
	}

	return err
}

func (w *FilesClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	return nil, fmt.Errorf("list API is not yet available for UC Volumes")
}

func (w *FilesClient) Mkdir(ctx context.Context, name string) error {
	// Directories are created implicitly.
	// No need to do anything.
	return nil
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
