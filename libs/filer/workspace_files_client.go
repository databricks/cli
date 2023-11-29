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
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/schollz/progressbar/v3"
)

// Type that implements fs.DirEntry for WSFS.
type wsfsDirEntry struct {
	wsfsFileInfo
}

func (entry wsfsDirEntry) Type() fs.FileMode {
	return entry.wsfsFileInfo.Mode()
}

func (entry wsfsDirEntry) Info() (fs.FileInfo, error) {
	return entry.wsfsFileInfo, nil
}

// Type that implements fs.FileInfo for WSFS.
type wsfsFileInfo struct {
	oi workspace.ObjectInfo
}

func (info wsfsFileInfo) Name() string {
	return path.Base(info.oi.Path)
}

func (info wsfsFileInfo) Size() int64 {
	return info.oi.Size
}

func (info wsfsFileInfo) Mode() fs.FileMode {
	switch info.oi.ObjectType {
	case workspace.ObjectTypeDirectory:
		return fs.ModeDir
	default:
		return fs.ModePerm
	}
}

func (info wsfsFileInfo) ModTime() time.Time {
	return time.UnixMilli(info.oi.ModifiedAt)
}

func (info wsfsFileInfo) IsDir() bool {
	return info.oi.ObjectType == workspace.ObjectTypeDirectory
}

func (info wsfsFileInfo) Sys() any {
	return info.oi
}

// WorkspaceFilesClient implements the files-in-workspace API.

// NOTE: This API is available for files under /Repos if a workspace has files-in-repos enabled.
// It can access any workspace path if files-in-workspace is enabled.
type WorkspaceFilesClient struct {
	workspaceClient *databricks.WorkspaceClient
	apiClient       *client.DatabricksClient

	// File operations will be relative to this path.
	root WorkspaceRootPath

	bar *progressbar.ProgressBar
}

func NewWorkspaceFilesClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}

	return &WorkspaceFilesClient{
		workspaceClient: w,
		apiClient:       apiClient,

		root: NewWorkspaceRootPath(root),
	}, nil
}

func NewWorkspaceFilesClientWithProgressLogging(w *databricks.WorkspaceClient, root string) (Filer, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}

	return &WorkspaceFilesClient{
		workspaceClient: w,
		apiClient:       apiClient,

		root: NewWorkspaceRootPath(root),

		bar: progressbar.DefaultBytes(100),
	}, nil
}

func (w *WorkspaceFilesClient) Write(ctx context.Context, name string, reader io.Reader, size int64, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Remove leading "/" so we can use it in the URL.
	overwrite := slices.Contains(mode, OverwriteIfExists)
	urlPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=%t",
		url.PathEscape(strings.TrimLeft(absPath, "/")),
		overwrite,
	)

	// Buffer the file contents because we may need to retry below and we cannot read twice.
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	var r io.Reader = bytes.NewBuffer(body)
	if w.bar != nil {
		w.bar.ChangeMax64(size)
		reader := progressbar.NewReader(r, w.bar)
		r = &reader
	}

	err = w.apiClient.Do(ctx, http.MethodPost, urlPath, nil, r, nil)

	// Return early on success.
	if err == nil {
		return nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return err
	}

	// This API returns a 404 if the parent directory does not exist.
	if aerr.StatusCode == http.StatusNotFound {
		if !slices.Contains(mode, CreateParentDirectories) {
			return NoSuchDirectoryError{path.Dir(absPath)}
		}

		// Create parent directory.
		err = w.workspaceClient.Workspace.MkdirsByPath(ctx, path.Dir(absPath))
		if err != nil {
			return fmt.Errorf("unable to mkdir to write file %s: %w", absPath, err)
		}

		// Retry without CreateParentDirectories mode flag.
		return w.Write(ctx, name, bytes.NewReader(body), int64(len(body)), sliceWithout(mode, CreateParentDirectories)...)
	}

	// This API returns 409 if the file already exists, when the object type is file
	if aerr.StatusCode == http.StatusConflict {
		return FileAlreadyExistsError{absPath}
	}

	// This API returns 400 if the file already exists, when the object type is notebook
	regex := regexp.MustCompile(`Path \((.*)\) already exists.`)
	if aerr.StatusCode == http.StatusBadRequest && regex.Match([]byte(aerr.Message)) {
		// Parse file path from regex capture group
		matches := regex.FindStringSubmatch(aerr.Message)
		if len(matches) == 2 {
			return FileAlreadyExistsError{matches[1]}
		}

		// Default to path specified to filer.Write if regex capture fails
		return FileAlreadyExistsError{absPath}
	}

	return err
}

func (w *WorkspaceFilesClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	// This stat call serves two purposes:
	// 1. Checks file at path exists, and throws an error if it does not
	// 2. Allows us to error out if the path is a directory. This is needed
	// because the /workspace/export API does not error out, and returns the directory
	// as a DBC archive even if format "SOURCE" is specified
	stat, err := w.Stat(ctx, name)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, NotAFile{absPath}
	}

	// Export file contents. Note the /workspace/export API has a limit of 10MBs
	// for the file size
	return w.workspaceClient.Workspace.Download(ctx, absPath)
}

func (w *WorkspaceFilesClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return CannotDeleteRootError{}
	}

	recursive := false
	if slices.Contains(mode, DeleteRecursively) {
		recursive = true
	}

	err = w.workspaceClient.Workspace.Delete(ctx, workspace.Delete{
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
		if aerr.ErrorCode == "DIRECTORY_NOT_EMPTY" {
			return DirectoryNotEmptyError{absPath}
		}
	case http.StatusNotFound:
		return FileDoesNotExistError{absPath}
	}

	return err
}

func (w *WorkspaceFilesClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	objects, err := w.workspaceClient.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
		Path: absPath,
	})

	if len(objects) == 1 && objects[0].Path == absPath {
		return nil, NotADirectory{absPath}
	}

	if err != nil {
		// If we got an API error we deal with it below.
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the specified path does not exist.
		if aerr.StatusCode == http.StatusNotFound {
			return nil, NoSuchDirectoryError{path.Dir(absPath)}
		}

		return nil, err
	}

	info := make([]fs.DirEntry, len(objects))
	for i, v := range objects {
		info[i] = wsfsDirEntry{wsfsFileInfo{oi: v}}
	}

	// Sort by name for parity with os.ReadDir.
	sort.Slice(info, func(i, j int) bool { return info[i].Name() < info[j].Name() })
	return info, nil
}

func (w *WorkspaceFilesClient) Mkdir(ctx context.Context, name string) error {
	dirPath, err := w.root.Join(name)
	if err != nil {
		return err
	}
	return w.workspaceClient.Workspace.Mkdirs(ctx, workspace.Mkdirs{
		Path: dirPath,
	})
}

func (w *WorkspaceFilesClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	info, err := w.workspaceClient.Workspace.GetStatusByPath(ctx, absPath)
	if err != nil {
		// If we got an API error we deal with it below.
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the specified path does not exist.
		if aerr.StatusCode == http.StatusNotFound {
			return nil, FileDoesNotExistError{absPath}
		}
	}

	return wsfsFileInfo{*info}, nil
}
