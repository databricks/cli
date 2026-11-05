package filer

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
)

// SandboxClient implements Filer using the sandbox Files APIs.
type SandboxClient struct {
	workspaceClient *databricks.WorkspaceClient
	apiClient       *client.DatabricksClient
	parent          string
	root            WorkspaceRootPath
}

// NewSandboxClient returns a filer rooted at an absolute path in a sandbox.
func NewSandboxClient(w *databricks.WorkspaceClient, sandboxID, root string) (Filer, error) {
	if sandboxID == "" || sandboxID == "." || sandboxID == ".." || strings.Contains(sandboxID, "/") {
		return nil, fmt.Errorf("invalid sandbox ID: %q", sandboxID)
	}
	if !path.IsAbs(root) {
		return nil, fmt.Errorf("sandbox root must be absolute: %s", root)
	}
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &SandboxClient{
		workspaceClient: w,
		apiClient:       apiClient,
		parent:          "sandboxes/" + url.PathEscape(sandboxID),
		root:            NewWorkspaceRootPath(root),
	}, nil
}

func (w *SandboxClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	if !slices.Contains(mode, CreateParentDirectories) {
		dir := path.Dir(absPath)
		info, err := w.Stat(ctx, path.Dir(name))
		if errors.Is(err, fs.ErrNotExist) {
			return noSuchDirectoryError{dir}
		}
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Mode()&fs.ModeSymlink == 0 {
			return notADirectory{dir}
		}
	}

	headers := map[string]string{"Content-Type": "application/octet-stream"}
	maps.Copy(headers, auth.WorkspaceIDHeaders(w.workspaceClient.Config))
	// The pinned SDK predates the overwrite parameter. Keep the body streaming.
	err = w.apiClient.Do(ctx, http.MethodPut,
		"/api/2.0/sandbox-files/"+w.parent+"/content", headers,
		map[string]any{"path": absPath, "overwrite": slices.Contains(mode, OverwriteIfExists)},
		io.NopCloser(reader), nil)
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok && aerr.StatusCode == http.StatusPreconditionFailed {
		return fileAlreadyExistsError{absPath}
	}
	return sandboxPathError(err, absPath)
}

func (w *SandboxClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}
	response, err := w.workspaceClient.Sandbox.DownloadFile(ctx, sandbox.DownloadFileRequest{
		Parent: w.parent,
		Path:   absPath,
	})
	if err != nil {
		return nil, sandboxPathError(err, absPath)
	}
	return response.Contents, nil
}

func (w *SandboxClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}
	if absPath == w.root.rootPath {
		return cannotDeleteRootError{}
	}
	// DeletePath is idempotent; Filer reports a missing entry.
	if _, err := w.Stat(ctx, name); err != nil {
		return err
	}
	_, err = w.workspaceClient.Sandbox.DeletePath(ctx, sandbox.DeletePathRequest{
		Parent:    w.parent,
		Path:      absPath,
		Recursive: slices.Contains(mode, DeleteRecursively),
	})
	return sandboxPathError(err, absPath)
}

func (w *SandboxClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}
	files, err := w.workspaceClient.Sandbox.ListDirectoryContentsAll(ctx, sandbox.ListDirectoryContentsRequest{
		Parent: w.parent,
		Path:   absPath,
	})
	if err != nil {
		return nil, sandboxPathError(err, absPath)
	}
	entries := make([]fs.DirEntry, len(files))
	for i, file := range files {
		entries[i] = fs.FileInfoToDirEntry(sandboxFileInfo{file})
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (w *SandboxClient) Mkdir(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}
	_, err = w.workspaceClient.Sandbox.CreateDirectory(ctx, sandbox.CreateDirectoryRequest{
		Parent:  w.parent,
		Path:    absPath,
		Parents: true,
	})
	return sandboxPathError(err, absPath)
}

func (w *SandboxClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}
	response, err := w.workspaceClient.Sandbox.GetMetadata(ctx, sandbox.GetMetadataRequest{
		Parent: w.parent,
		Path:   absPath,
	})
	if err != nil {
		return nil, sandboxPathError(err, absPath)
	}
	return sandboxFileInfo{*response.File}, nil
}

func sandboxPathError(err error, name string) error {
	aerr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return err
	}
	switch aerr.StatusCode {
	case http.StatusNotFound:
		return fileDoesNotExistError{name}
	case http.StatusForbidden:
		return permissionError{name, err}
	default:
		return err
	}
}

type sandboxFileInfo struct {
	fi sandbox.SandboxFileInfo
}

func (info sandboxFileInfo) Name() string { return path.Base(info.fi.Path) }
func (info sandboxFileInfo) Size() int64  { return info.fi.Size }
func (info sandboxFileInfo) Sys() any     { return info.fi }

func (info sandboxFileInfo) IsDir() bool {
	return info.fi.FileType == sandbox.SandboxFileTypeSandboxFileTypeDirectory
}

func (info sandboxFileInfo) Mode() fs.FileMode {
	mode := fs.FileMode(info.fi.Mode) & fs.ModePerm
	switch info.fi.FileType {
	case sandbox.SandboxFileTypeSandboxFileTypeDirectory:
		mode |= fs.ModeDir
	case sandbox.SandboxFileTypeSandboxFileTypeSymlink:
		mode |= fs.ModeSymlink
	case sandbox.SandboxFileTypeSandboxFileTypeFile:
	default:
		mode |= fs.ModeIrregular
	}
	return mode
}

func (info sandboxFileInfo) ModTime() time.Time {
	return info.fi.ModifyTime.AsTime()
}
