package filer

import (
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
	"golang.org/x/sync/errgroup"
)

// As of 19th Feb 2024, the Files API backend has a rate limit of 10 concurrent
// requests and 100 QPS. We limit the number of concurrent requests to 5 to
// avoid hitting the rate limit.
const maxFilesRequestsInFlight = 5

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
	urlPath := "/api/2.0/fs/files/" + url.PathEscape(strings.TrimLeft(absPath, "/"))

	return absPath, urlPath, nil
}

func (w *FilesClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, urlPath, err := w.urlPath(name)
	if err != nil {
		return err
	}

	// Check that target path exists if CreateParentDirectories mode is not set
	if !slices.Contains(mode, CreateParentDirectories) {
		err := w.workspaceClient.Files.GetDirectoryMetadataByDirectoryPath(ctx, path.Dir(absPath))
		if err != nil {
			var aerr *apierr.APIError
			if !errors.As(err, &aerr) {
				return err
			}

			// This API returns a 404 if the file doesn't exist.
			if aerr.StatusCode == http.StatusNotFound {
				return NoSuchDirectoryError{path.Dir(absPath)}
			}

			return err
		}
	}

	overwrite := slices.Contains(mode, OverwriteIfExists)
	urlPath = fmt.Sprintf("%s?overwrite=%t", urlPath, overwrite)
	headers := map[string]string{"Content-Type": "application/octet-stream"}
	err = w.apiClient.Do(ctx, http.MethodPut, urlPath, headers, nil, reader, nil)

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
	if aerr.StatusCode == http.StatusConflict && aerr.ErrorCode == "ALREADY_EXISTS" {
		return FileAlreadyExistsError{absPath}
	}

	return err
}

func (w *FilesClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, urlPath, err := w.urlPath(name)
	if err != nil {
		return nil, err
	}

	var reader io.ReadCloser
	err = w.apiClient.Do(ctx, http.MethodGet, urlPath, nil, nil, nil, &reader)

	// Return early on success.
	if err == nil {
		return reader, nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return nil, err
	}

	// This API returns a 404 if the specified path does not exist.
	if aerr.StatusCode == http.StatusNotFound {
		// Check if the path is a directory. If so, return not a file error.
		if _, err := w.statDir(ctx, name); err == nil {
			return nil, NotAFile{absPath}
		}

		// No file or directory exists at the specified path. Return no such file error.
		return nil, FileDoesNotExistError{absPath}
	}

	return nil, err
}

func (w *FilesClient) deleteFile(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return CannotDeleteRootError{}
	}

	err = w.workspaceClient.Files.DeleteByFilePath(ctx, absPath)

	// Return early on success.
	if err == nil {
		return nil
	}

	var aerr *apierr.APIError
	// Special handling of this error only if it is an API error.
	if !errors.As(err, &aerr) {
		return err
	}

	// This files delete API returns a 404 if the specified path does not exist.
	if aerr.StatusCode == http.StatusNotFound {
		return FileDoesNotExistError{absPath}
	}

	return err
}

func (w *FilesClient) deleteDirectory(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return CannotDeleteRootError{}
	}

	err = w.workspaceClient.Files.DeleteDirectoryByDirectoryPath(ctx, absPath)

	var aerr *apierr.APIError
	// Special handling of this error only if it is an API error.
	if !errors.As(err, &aerr) {
		return err
	}

	// The directory delete API returns a 400 if the directory is not empty
	if aerr.StatusCode == http.StatusBadRequest {
		var reasons []string
		details := aerr.ErrorDetails()
		if details.ErrorInfo != nil {
			reasons = append(reasons, details.ErrorInfo.Reason)
		}
		// Error code 400 is generic and can be returned for other reasons. Make
		// sure one of the reasons for the error is that the directory is not empty.
		if !slices.Contains(reasons, "FILES_API_DIRECTORY_IS_NOT_EMPTY") {
			return err
		}
		return DirectoryNotEmptyError{absPath}
	}

	return err
}

func (w *FilesClient) recursiveDelete(ctx context.Context, name string) error {
	filerFS := NewFS(ctx, w)
	var dirsToDelete []string
	var filesToDelete []string
	callback := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Files API does not allowing deleting non-empty directories. We instead
		// collect the directories to delete and delete them once all the files have
		// been deleted.
		if d.IsDir() {
			dirsToDelete = append(dirsToDelete, path)
			return nil
		}

		filesToDelete = append(filesToDelete, path)
		return nil
	}

	// Walk the directory and accumulate the files and directories to delete.
	err := fs.WalkDir(filerFS, name, callback)
	if err != nil {
		return err
	}

	// Delete the files in parallel.
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(maxFilesRequestsInFlight)

	for _, file := range filesToDelete {
		// Skip the file if the context has already been cancelled.
		select {
		case <-groupCtx.Done():
			continue
		default:
			// Proceed.
		}

		group.Go(func() error {
			return w.deleteFile(groupCtx, file)
		})
	}

	// Wait for the files to be deleted and return the first non-nil error.
	err = group.Wait()
	if err != nil {
		return err
	}

	// Delete the directories in reverse order to ensure that the parent
	// directories are deleted after the children. This is possible because
	// fs.WalkDir walks the directories in lexicographical order.
	for i := len(dirsToDelete) - 1; i >= 0; i-- {
		err := w.deleteDirectory(ctx, dirsToDelete[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *FilesClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	if slices.Contains(mode, DeleteRecursively) {
		return w.recursiveDelete(ctx, name)
	}

	// Issue a stat call to determine if the path is a file or directory.
	info, err := w.Stat(ctx, name)
	if err != nil {
		return err
	}

	// Issue the delete call for a directory
	if info.IsDir() {
		return w.deleteDirectory(ctx, name)
	}

	return w.deleteFile(ctx, name)
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

	// Return early on success.
	if err == nil {
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

	// Special handling of this error only if it is an API error.
	var apierr *apierr.APIError
	if !errors.As(err, &apierr) {
		return nil, err
	}

	// This API returns a 404 if the specified path does not exist.
	if apierr.StatusCode == http.StatusNotFound {
		// Check if the path is a file. If so, return not a directory error.
		if _, err := w.statFile(ctx, name); err == nil {
			return nil, NotADirectory{absPath}
		}

		// No file or directory exists at the specified path. Return no such directory error.
		return nil, NoSuchDirectoryError{absPath}
	}
	return nil, err
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

// Get file metadata for a file using the Files API.
func (w *FilesClient) statFile(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	fileInfo, err := w.workspaceClient.Files.GetMetadataByFilePath(ctx, absPath)

	// If the HEAD requests succeeds, the file exists.
	if err == nil {
		return filesApiFileInfo{
			absPath:  absPath,
			isDir:    false,
			fileSize: fileInfo.ContentLength,
		}, nil
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

// Get file metadata for a directory using the Files API.
func (w *FilesClient) statDir(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	err = w.workspaceClient.Files.GetDirectoryMetadataByDirectoryPath(ctx, absPath)

	// If the HEAD requests succeeds, the directory exists.
	if err == nil {
		return filesApiFileInfo{absPath: absPath, isDir: true}, nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return nil, err
	}

	// The directory metadata API returns a 404 if the specified path does not exist.
	if aerr.StatusCode == http.StatusNotFound {
		return nil, NoSuchDirectoryError{absPath}
	}

	return nil, err
}

func (w *FilesClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	// Assume that the path is a directory and issue a stat call.
	dirInfo, err := w.statDir(ctx, name)

	// If the file exists, return early.
	if err == nil {
		return dirInfo, nil
	}

	// Return early if the error is not a NoSuchDirectoryError.
	if !errors.As(err, &NoSuchDirectoryError{}) {
		return nil, err
	}

	// Since the path is not a directory, assume that it is a file and issue a stat call.
	return w.statFile(ctx, name)
}
