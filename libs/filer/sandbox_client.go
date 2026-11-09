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

// sandboxStatus returns the HTTP status code of err if it is (or wraps) an
// [apierr.APIError], and -1 otherwise.
//
// Sandbox filesystem errors carry the same reason and message ("sandbox
// filesystem operation failed") for unrelated conditions, and their error codes
// don't always match the status (a download of a directory is 409 with error
// code BAD_REQUEST). The status is the most specific signal, but it is still
// ambiguous, so callers stat the path to confirm a condition before mapping it.
func sandboxStatus(err error) int {
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok {
		return aerr.StatusCode
	}
	return -1
}

// sandboxPathError maps the errors that mean the same for every operation. Callers map
// operation-specific statuses first, because those need a stat to disambiguate.
func sandboxPathError(err error, absPath string) error {
	switch {
	case errors.Is(err, apierr.ErrNotFound):
		return fileDoesNotExistError{absPath}
	case sandboxStatus(err) == http.StatusForbidden:
		return permissionError{absPath, err}
	default:
		return err
	}
}

// sandboxFileInfo implements [fs.FileInfo] for an entry in a sandbox filesystem.
type sandboxFileInfo struct {
	info sandbox.SandboxFileInfo
}

func (i sandboxFileInfo) Name() string {
	return i.info.Name
}

func (i sandboxFileInfo) Size() int64 {
	return i.info.Size
}

// Mode reports symlinks as [fs.ModeSymlink] without following them, matching
// [os.Lstat]: the metadata and listing APIs describe the link itself, and
// reporting the link's own type keeps recursive walks from following links out
// of the tree. Read and Write still follow links, because the server does
// (per the API docs for download and upload).
// Only the permission bits of the Unix mode are kept; setuid, setgid and sticky
// use different bits in [fs.FileMode].
func (i sandboxFileInfo) Mode() fs.FileMode {
	mode := fs.FileMode(i.info.Mode) & fs.ModePerm
	switch i.info.FileType {
	case sandbox.SandboxFileTypeSandboxFileTypeFile:
	case sandbox.SandboxFileTypeSandboxFileTypeDirectory:
		mode |= fs.ModeDir
	case sandbox.SandboxFileTypeSandboxFileTypeSymlink:
		mode |= fs.ModeSymlink
	case sandbox.SandboxFileTypeSandboxFileTypeOther:
		mode |= fs.ModeIrregular
	}
	return mode
}

func (i sandboxFileInfo) ModTime() time.Time {
	return i.info.ModifyTime.AsTime()
}

func (i sandboxFileInfo) IsDir() bool {
	return i.info.FileType == sandbox.SandboxFileTypeSandboxFileTypeDirectory
}

func (i sandboxFileInfo) Sys() any {
	return i.info
}

// SandboxClient implements the [Filer] interface for the filesystem of a
// single Databricks sandbox.
//
// Uploads create missing parents and deletes succeed for absent paths, so this
// client checks those paths to enforce the stricter Filer contract.
type SandboxClient struct {
	api          sandbox.SandboxInterface
	uploadClient *client.DatabricksClient

	// parent is the sandbox resource name, "sandboxes/{sandbox_id}", that every
	// file API call is scoped to.
	parent string

	// File operations will be relative to this path.
	root WorkspaceRootPath
}

// NewSandboxClient returns a [Filer] for the filesystem of sandbox sandboxID,
// rooted at root, an absolute path inside the sandbox. Names are joined to root,
// and a name that would resolve outside it is rejected (see
// [WorkspaceRootPath.Join]). Per the API docs, a stopped sandbox starts
// automatically on the first call, which makes that call slow.
func NewSandboxClient(w *databricks.WorkspaceClient, sandboxID, root string) (Filer, error) {
	// The ID becomes a path segment of every request URL, so it can't contain "/" or be a
	// relative segment that the URL would resolve away.
	if sandboxID == "" || sandboxID == "." || sandboxID == ".." || strings.Contains(sandboxID, "/") {
		return nil, fmt.Errorf("invalid sandbox ID: %q", sandboxID)
	}
	if !path.IsAbs(root) {
		return nil, fmt.Errorf("sandbox root must be absolute: %s", root)
	}
	uploadClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &SandboxClient{
		api:          w.Sandbox,
		uploadClient: uploadClient,
		parent:       "sandboxes/" + url.PathEscape(sandboxID),
		root:         NewWorkspaceRootPath(root),
	}, nil
}

func (c *SandboxClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := c.root.Join(name)
	if err != nil {
		return err
	}

	// The server always creates missing parent directories, but the Filer
	// contract requires fs.ErrNotExist without CreateParentDirectories. A parent
	// created or removed after this check only changes which error is returned.
	if !slices.Contains(mode, CreateParentDirectories) {
		dir := path.Dir(absPath)
		info, err := c.stat(ctx, dir)
		if errors.Is(err, fs.ErrNotExist) {
			return noSuchDirectoryError{dir}
		}
		if err != nil {
			return err
		}
		// stat doesn't follow a final symlink, but upload does, so a symlink to a
		// directory is a valid parent. A symlink to a file fails the upload instead.
		if !info.IsDir() && info.Mode()&fs.ModeSymlink == 0 {
			return notADirectory{dir}
		}
	}

	overwrite := slices.Contains(mode, OverwriteIfExists)
	if !overwrite {
		// Avoid streaming skipped files; overwrite=false still protects concurrent creates.
		_, err = c.stat(ctx, absPath)
		if err == nil {
			return fileAlreadyExistsError{absPath}
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	headers := map[string]string{"Content-Type": "application/octet-stream"}
	maps.Copy(headers, auth.WorkspaceIDHeaders(c.uploadClient.Config))
	// The pinned SDK lacks the overwrite field; its HTTP client still streams the body.
	err = c.uploadClient.Do(ctx, http.MethodPut, "/api/2.0/sandbox-files/"+c.parent+"/content",
		headers, map[string]any{"path": absPath, "overwrite": overwrite}, io.NopCloser(reader), nil)
	if !overwrite && sandboxStatus(err) == http.StatusPreconditionFailed {
		return fileAlreadyExistsError{absPath}
	}
	// A 409 can indicate a directory target or a parent component that is a file.
	if sandboxStatus(err) == http.StatusConflict {
		if info, serr := c.stat(ctx, absPath); serr == nil && info.IsDir() {
			return notAFile{absPath}
		}
	}
	return sandboxPathError(err, absPath)
}

func (c *SandboxClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := c.root.Join(name)
	if err != nil {
		return nil, err
	}

	resp, err := c.api.DownloadFile(ctx, sandbox.DownloadFileRequest{
		Parent: c.parent,
		Path:   absPath,
	})
	if err == nil {
		return resp.Contents, nil
	}

	// The server returns 409 for a directory, and also when a parent path
	// component is a file. Stat to tell them apart. Download follows symlinks
	// but stat doesn't, so a 409 for a symlink means it points at a directory.
	if sandboxStatus(err) == http.StatusConflict {
		if info, serr := c.stat(ctx, absPath); serr == nil && (info.IsDir() || info.Mode()&fs.ModeSymlink != 0) {
			return nil, notAFile{absPath}
		}
	}

	// The server returns 404 for a missing file or missing parent directory. It
	// also returns 404 for an unknown sandbox, with the same message ("The
	// sandbox or file was not found."), so that is reported as a missing file too.
	return nil, sandboxPathError(err, absPath)
}

func (c *SandboxClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := c.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == c.root.rootPath {
		return cannotDeleteRootError{}
	}

	// The server succeeds if the path is already absent, but the Filer contract
	// requires fs.ErrNotExist. The stat also tells a directory from a file, which
	// the error mapping below needs.
	info, err := c.stat(ctx, absPath)
	if err != nil {
		return err
	}

	// A recursive delete is a single server-side call, unlike the Files API
	// filer, which walks the tree.
	recursive := slices.Contains(mode, DeleteRecursively)
	_, err = c.api.DeletePath(ctx, sandbox.DeletePathRequest{
		Parent:    c.parent,
		Path:      absPath,
		Recursive: recursive,
	})

	// The server returns a generic 400 for a non-empty directory without the
	// recursive flag. The stat above confirmed that the path is a directory.
	if !recursive && info.IsDir() && sandboxStatus(err) == http.StatusBadRequest {
		return directoryNotEmptyError{absPath}
	}

	return sandboxPathError(err, absPath)
}

func (c *SandboxClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := c.root.Join(name)
	if err != nil {
		return nil, err
	}

	// The listing is paginated (up to 1000 entries per page); this collects all pages.
	infos, err := c.api.ListDirectoryContentsAll(ctx, sandbox.ListDirectoryContentsRequest{
		Parent: c.parent,
		Path:   absPath,
	})
	if err != nil {
		if errors.Is(err, apierr.ErrNotFound) {
			return nil, noSuchDirectoryError{absPath}
		}

		// The server returns a generic 400 when the path is a file.
		if sandboxStatus(err) == http.StatusBadRequest {
			if info, serr := c.stat(ctx, absPath); serr == nil && !info.IsDir() {
				return nil, notADirectory{absPath}
			}
		}

		return nil, sandboxPathError(err, absPath)
	}

	entries := make([]fs.DirEntry, len(infos))
	for i, info := range infos {
		entries[i] = fs.FileInfoToDirEntry(sandboxFileInfo{info})
	}
	// Sort for parity with os.ReadDir instead of relying on the server's
	// documented name order.
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (c *SandboxClient) Mkdir(ctx context.Context, name string) error {
	absPath, err := c.root.Join(name)
	if err != nil {
		return err
	}

	_, err = c.api.CreateDirectory(ctx, sandbox.CreateDirectoryRequest{
		Parent: c.parent,
		Path:   absPath,
		// Filer.Mkdir creates intermediate directories. With parents=false,
		// the server also returns 409 if the directory already exists.
		Parents: true,
	})

	// The server returns a generic 400 when a file exists at the path.
	if sandboxStatus(err) == http.StatusBadRequest {
		if info, serr := c.stat(ctx, absPath); serr == nil && !info.IsDir() {
			return fileAlreadyExistsError{absPath}
		}
	}

	return sandboxPathError(err, absPath)
}

func (c *SandboxClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := c.root.Join(name)
	if err != nil {
		return nil, err
	}

	return c.stat(ctx, absPath)
}

// stat describes absPath without following a final symlink; see
// [sandboxFileInfo.Mode] for why that is kept.
func (c *SandboxClient) stat(ctx context.Context, absPath string) (fs.FileInfo, error) {
	resp, err := c.api.GetMetadata(ctx, sandbox.GetMetadataRequest{
		Parent: c.parent,
		Path:   absPath,
	})
	if err != nil {
		return nil, sandboxPathError(err, absPath)
	}

	return sandboxFileInfo{*resp.File}, nil
}
