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
	"sort"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/exp/slices"
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
	return nil
}

// WorkspaceFilesClient implements the files-in-workspace API.

// NOTE: This API is available for files under /Repos if a workspace has files-in-repos enabled.
// It can access any workspace path if files-in-workspace is enabled.
type WorkspaceFilesClient struct {
	workspaceClient *databricks.WorkspaceClient
	apiClient       *client.DatabricksClient

	// File operations will be relative to this path.
	root RootPath
}

func NewWorkspaceFilesClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}

	return &WorkspaceFilesClient{
		workspaceClient: w,
		apiClient:       apiClient,

		root: NewRootPath(root),
	}, nil
}

func (w *WorkspaceFilesClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
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

	err = w.apiClient.Do(ctx, http.MethodPost, urlPath, body, nil)

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
		return w.Write(ctx, name, bytes.NewReader(body), sliceWithout(mode, CreateParentDirectories)...)
	}

	// This API returns 409 if the file already exists.
	if aerr.StatusCode == http.StatusConflict {
		return FileAlreadyExistsError{absPath}
	}

	return err
}

func (w *WorkspaceFilesClient) Read(ctx context.Context, name string) (io.Reader, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	// Remove leading "/" so we can use it in the URL.
	urlPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(absPath, "/"),
	)

	var res []byte
	err = w.apiClient.Do(ctx, http.MethodGet, urlPath, nil, &res)

	// Return early on success.
	if err == nil {
		return bytes.NewReader(res), nil
	}

	// Special handling of this error only if it is an API error.
	var aerr *apierr.APIError
	if !errors.As(err, &aerr) {
		return nil, err
	}

	if aerr.StatusCode == http.StatusNotFound {
		return nil, FileDoesNotExistError{absPath}
	}

	return nil, err
}

func (w *WorkspaceFilesClient) Delete(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	err = w.workspaceClient.Workspace.Delete(ctx, workspace.Delete{
		Path:      absPath,
		Recursive: false,
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

	if aerr.StatusCode == http.StatusNotFound {
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

	// TODO: add integration test for this
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
