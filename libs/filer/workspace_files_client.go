package filer

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// Type that implements fs.DirEntry for WSFS.
type wsfsDirEntry struct {
	wsfsFileInfo
}

func (entry wsfsDirEntry) Type() fs.FileMode {
	return entry.Mode()
}

func (entry wsfsDirEntry) Info() (fs.FileInfo, error) {
	return entry.wsfsFileInfo, nil
}

func wsfsDirEntriesFromObjectInfos(objects []workspace.ObjectInfo) []fs.DirEntry {
	info := make([]fs.DirEntry, len(objects))
	for i, v := range objects {
		info[i] = wsfsDirEntry{wsfsFileInfo{ObjectInfo: v}}
	}

	// Sort by name for parity with os.ReadDir.
	slices.SortFunc(info, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return info
}

// Type that implements fs.FileInfo for WSFS.
type wsfsFileInfo struct { //nolint:recvcheck // value receivers for fs.FileInfo interface, pointer for JSON marshaling
	workspace.ObjectInfo

	// The export format of a notebook. This is not exposed by the SDK.
	ReposExportFormat workspace.ExportFormat `json:"repos_export_format,omitempty"`
}

func (info wsfsFileInfo) Name() string {
	return path.Base(info.Path)
}

func (info wsfsFileInfo) Size() int64 {
	return info.ObjectInfo.Size
}

func (info wsfsFileInfo) Mode() fs.FileMode {
	switch info.ObjectType {
	case workspace.ObjectTypeDirectory, workspace.ObjectTypeRepo:
		return fs.ModeDir
	default:
		return fs.ModePerm
	}
}

func (info wsfsFileInfo) ModTime() time.Time {
	return time.UnixMilli(info.ModifiedAt)
}

func (info wsfsFileInfo) IsDir() bool {
	return info.Mode() == fs.ModeDir
}

func (info wsfsFileInfo) Sys() any {
	return info.ObjectInfo
}

func (info wsfsFileInfo) WorkspaceObjectInfo() workspace.ObjectInfo {
	return info.ObjectInfo
}

// UnmarshalJSON is a custom unmarshaller for the wsfsFileInfo struct.
// It must be defined for this type because otherwise the implementation
// of the embedded ObjectInfo type will be used.
func (info *wsfsFileInfo) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, info)
}

// MarshalJSON is a custom marshaller for the wsfsFileInfo struct.
// It must be defined for this type because otherwise the implementation
// of the embedded ObjectInfo type will be used.
func (info *wsfsFileInfo) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(info)
}

// Interface for *client.DatabricksClient from the Databricks Go SDK. Abstracted
// as an interface to allow for mocking in tests.
type apiClient interface {
	Do(ctx context.Context, method, path string,
		headers map[string]string, queryString map[string]any, request, response any,
		visitors ...func(*http.Request) error) error
}

// WorkspaceFilesClient implements the files-in-workspace API.

// NOTE: This API is available for files under /Repos if a workspace has files-in-repos enabled.
// It can access any workspace path if files-in-workspace is enabled.
type WorkspaceFilesClient struct {
	workspaceClient *databricks.WorkspaceClient
	apiClient       apiClient

	// File operations will be relative to this path.
	root WorkspaceRootPath
}

// orgIDHeaders returns headers with X-Databricks-Org-Id set if a workspace ID
// is configured. SPOG hosts require this header to route requests to the
// correct workspace.
func (w *WorkspaceFilesClient) orgIDHeaders() map[string]string {
	if w.workspaceClient == nil || w.workspaceClient.Config == nil {
		return nil
	}
	wsID := w.workspaceClient.Config.WorkspaceID
	if wsID == "" {
		return nil
	}
	return map[string]string{
		"X-Databricks-Org-Id": wsID,
	}
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

func (w *WorkspaceFilesClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Buffer the file contents because we may need to retry below and we cannot read twice.
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	// Use Workspace.Upload (multipart /api/2.0/workspace/import) instead of the
	// JSON-body variant of the same endpoint, which caps payloads at 10 MiB for
	// AUTO format (databricks.webapp.autoExportFormatLimitBytes). The multipart
	// variant has been verified against a real workspace at 450 MB for regular
	// files — strictly better than the legacy /workspace-files/import-file
	// endpoint we are migrating away from, which has a 200 MiB body cap
	// (databricks.workspaceFilesystem.maxImportSizeBytes) plus a 305s server-side
	// request timeout that cuts off uploads above ~400 MB at typical bandwidth.
	//
	// Notebook content (any payload with a `# Databricks notebook source` header
	// detected by format=AUTO) hits a separate 10 MiB cap on the server
	// (databricks.notebook.maxNotebookSizeBytes); this cap is enforced regardless
	// of which upload endpoint we use, so it is not a regression introduced by
	// this migration.
	overwrite := slices.Contains(mode, OverwriteIfExists)
	uploadOpts := []func(*workspace.Import){
		workspace.UploadFormat(workspace.ImportFormatAuto),
	}
	if overwrite {
		uploadOpts = append(uploadOpts, workspace.UploadOverwrite())
	}
	err = w.workspaceClient.Workspace.Upload(ctx, absPath, bytes.NewReader(body), uploadOpts...)

	// Return early on success.
	if err == nil {
		return nil
	}

	// Parent directory does not exist.
	if errors.Is(err, apierr.ErrNotFound) {
		if !slices.Contains(mode, CreateParentDirectories) {
			return noSuchDirectoryError{path.Dir(absPath)}
		}

		// Create parent directory.
		err = w.workspaceClient.Workspace.MkdirsByPath(ctx, path.Dir(absPath)) //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
		if err != nil {
			if errors.Is(err, apierr.ErrPermissionDenied) {
				return permissionError{absPath}
			}
			return fmt.Errorf("unable to mkdir to write file %s: %w", absPath, err)
		}

		// Retry without CreateParentDirectories mode flag.
		return w.Write(ctx, name, bytes.NewReader(body), sliceWithout(mode, CreateParentDirectories)...)
	}

	// File already exists at the path. The /workspace/import endpoint reports this
	// with two different error_codes depending on whether the conflict was detected
	// sequentially (400 RESOURCE_ALREADY_EXISTS) or under concurrent contention
	// (409 ALREADY_EXISTS, observed in TestLock). Both are already-exists from the
	// caller's perspective.
	//
	// Existing-object-with-mismatched-node-type (e.g. uploading a regular .py when a
	// NOTEBOOK is at the path) surfaces as 400 INVALID_PARAMETER_VALUE with a
	// "Requested node type" message — also already-exists from the caller's perspective.
	if errors.Is(err, apierr.ErrResourceAlreadyExists) || errors.Is(err, apierr.ErrAlreadyExists) {
		return fileAlreadyExistsError{absPath}
	}
	if errors.Is(err, apierr.ErrInvalidParameterValue) {
		var aerr *apierr.APIError
		if errors.As(err, &aerr) && strings.Contains(aerr.Message, "Requested node type") {
			return fileAlreadyExistsError{absPath}
		}
	}

	// Caller has read access but no write access.
	if errors.Is(err, apierr.ErrPermissionDenied) {
		return permissionError{absPath}
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
		return nil, notAFile{absPath}
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
		return cannotDeleteRootError{}
	}

	recursive := slices.Contains(mode, DeleteRecursively)

	err = w.workspaceClient.Workspace.Delete(ctx, workspace.Delete{ //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
		Path:      absPath,
		Recursive: recursive,
	})

	// Return early on success.
	if err == nil {
		return nil
	}

	if errors.Is(err, apierr.ErrNotFound) {
		return fileDoesNotExistError{absPath}
	}

	// No SDK sentinel for DIRECTORY_NOT_EMPTY; match the error_code directly.
	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.ErrorCode == "DIRECTORY_NOT_EMPTY" {
		return directoryNotEmptyError{absPath}
	}

	return err
}

func (w *WorkspaceFilesClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	objects, err := w.workspaceClient.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{ //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
		Path: absPath,
	})

	if len(objects) == 1 && objects[0].Path == absPath {
		return nil, notADirectory{absPath}
	}

	if err != nil {
		// NOTE: This API returns a 404 if the specified path does not exist,
		// but can also do so if we don't have read access.
		if errors.Is(err, apierr.ErrNotFound) {
			return nil, noSuchDirectoryError{path.Dir(absPath)}
		}
		return nil, err
	}

	// Convert to fs.DirEntry.
	return wsfsDirEntriesFromObjectInfos(objects), nil
}

func (w *WorkspaceFilesClient) Mkdir(ctx context.Context, name string) error {
	dirPath, err := w.root.Join(name)
	if err != nil {
		return err
	}
	return w.workspaceClient.Workspace.Mkdirs(ctx, workspace.Mkdirs{ //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
		Path: dirPath,
	})
}

func (w *WorkspaceFilesClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	var stat wsfsFileInfo

	// Perform bespoke API call because "return_export_info" is not exposed by the SDK.
	// We need "repos_export_format" to determine if the file is a py or a ipynb notebook.
	// This is not exposed by the SDK so we need to make a direct API call.
	err = w.apiClient.Do(
		ctx,
		http.MethodGet,
		"/api/2.0/workspace/get-status",
		w.orgIDHeaders(),
		nil,
		map[string]string{
			"path":               absPath,
			"return_export_info": "true",
		},
		&stat,
	)
	if err != nil {
		// This API returns a 404 if the specified path does not exist.
		if errors.Is(err, apierr.ErrNotFound) {
			return nil, fileDoesNotExistError{absPath}
		}
	}

	return stat, nil
}
