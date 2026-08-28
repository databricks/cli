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

	"github.com/databricks/cli/libs/auth"
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

// workspaceIDHeaders returns the workspace routing header map for outbound
// API calls, or nil if the workspace client is unset. Wraps the shared
// auth.WorkspaceIDHeaders helper with a nil-safe workspaceClient guard
// since this filer struct can legitimately be constructed without one in
// some test setups.
func (w *WorkspaceFilesClient) workspaceIDHeaders() map[string]string {
	if w.workspaceClient == nil {
		return nil
	}
	return auth.WorkspaceIDHeaders(w.workspaceClient.Config)
}

func NewWorkspaceFilesClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	// Workspace.Upload/Download forward cfg.WorkspaceID behind a bare != "" check,
	// so normalize the "none" sentinel to "" to keep the SDK from routing on it.
	if w.Config.WorkspaceID == auth.WorkspaceIDNone {
		w.Config.WorkspaceID = ""
	}

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

	// Upload with the SDK's multipart Workspace.Upload; format=AUTO lets the server
	// classify each payload as file or notebook. The JSON-body workspace.Import caps
	// content at 10 MB, the multipart form at the 500 MB workspace file limit.
	opts := []workspace.UploadOption{workspace.UploadFormat(workspace.ImportFormatAuto)}
	if slices.Contains(mode, OverwriteIfExists) {
		opts = append(opts, workspace.UploadOverwrite())
	}

	err = w.workspaceClient.Workspace.Upload(ctx, absPath, bytes.NewReader(body), opts...)

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
		err = w.workspaceClient.Workspace.MkdirsByPath(ctx, path.Dir(absPath))
		if err != nil {
			if errors.Is(err, apierr.ErrPermissionDenied) {
				return permissionError{absPath, err}
			}
			return fmt.Errorf("unable to mkdir to write file %s: %w", absPath, err)
		}

		// Retry without CreateParentDirectories mode flag.
		return w.Write(ctx, name, bytes.NewReader(body), sliceWithout(mode, CreateParentDirectories)...)
	}

	// Path already taken. ErrResourceConflict covers every 409 (including the
	// bare 409 with only a message); ErrResourceAlreadyExists covers the 400
	// RESOURCE_ALREADY_EXISTS, which no status maps to a conflict.
	if errors.Is(err, apierr.ErrResourceConflict) || errors.Is(err, apierr.ErrResourceAlreadyExists) {
		return fileAlreadyExistsError{absPath}
	}
	// Overwrite rejected because the existing object's node type differs from
	// the upload. The collision carries an AIP-193 ErrorInfo with a stable
	// reason, so branch on it rather than the message text.
	if errors.Is(err, apierr.ErrInvalidParameterValue) {
		if aerr, ok := errors.AsType[*apierr.APIError](err); ok {
			if info := aerr.ErrorDetails().ErrorInfo; info != nil && info.Reason == "WORKSPACE_OBJECT_TYPE_MISMATCH" {
				return fileAlreadyExistsError{absPath}
			}
		}
	}

	// A notebook that already exists returns 400 with an empty error_code, which
	// unwraps to ErrBadRequest by status alone, so this runs after the
	// ErrInvalidParameterValue branch. Anchor on the shared "already exists."
	// marker rather than parsing the full message.
	if errors.Is(err, apierr.ErrBadRequest) {
		if aerr, ok := errors.AsType[*apierr.APIError](err); ok {
			if strings.Contains(aerr.Message, "already exists.") {
				return fileAlreadyExistsError{absPath}
			}
		}
	}

	// Caller has read access but no write access.
	if errors.Is(err, apierr.ErrPermissionDenied) {
		return permissionError{absPath, err}
	}

	// Any other failure (e.g. a server-side error with an empty message) is
	// surfaced with the target path so the user can tell which upload failed;
	// %w keeps the original error inspectable by callers.
	return fmt.Errorf("failed to upload %s: %w", absPath, err)
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

	err = w.workspaceClient.Workspace.Delete(ctx, workspace.Delete{
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
		if aerr.ErrorCode == "DIRECTORY_NOT_EMPTY" {
			return directoryNotEmptyError{absPath}
		}
	case http.StatusNotFound:
		return fileDoesNotExistError{absPath}
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
		return nil, notADirectory{absPath}
	}

	if err != nil {
		// If we got an API error we deal with it below.
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// NOTE: This API returns a 404 if the specified path does not exist,
		// but can also do so if we don't have read access.
		if aerr.StatusCode == http.StatusNotFound {
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
	return w.workspaceClient.Workspace.Mkdirs(ctx, workspace.Mkdirs{
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
		w.workspaceIDHeaders(),
		nil,
		map[string]string{
			"path":               absPath,
			"return_export_info": "true",
		},
		&stat,
	)
	if err != nil {
		// If we got an API error we deal with it below.
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// This API returns a 404 if the specified path does not exist.
		if aerr.StatusCode == http.StatusNotFound {
			return nil, fileDoesNotExistError{absPath}
		}

		return nil, err
	}

	return stat, nil
}
