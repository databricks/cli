package filer

import (
	"cmp"
	"context"
	"crypto/rand"
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
// The sandbox file APIs are more permissive than the Filer contract: uploads
// always replace the target and create missing parents, and deletes succeed if
// the path is absent. Where the contract is stricter, this client checks the
// path first or, for exclusive creates, uses rename (see createExclusive).
type SandboxClient struct {
	api sandbox.SandboxInterface

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
	return &SandboxClient{
		api:    w.Sandbox,
		parent: "sandboxes/" + url.PathEscape(sandboxID),
		root:   NewWorkspaceRootPath(root),
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

	if slices.Contains(mode, OverwriteIfExists) {
		err := c.upload(ctx, absPath, reader)
		// The server returns 409 when the target is a directory, and also when a
		// parent path component is a file. Stat to tell them apart.
		if sandboxStatus(err) == http.StatusConflict {
			if info, serr := c.stat(ctx, absPath); serr == nil && info.IsDir() {
				return notAFile{absPath}
			}
		}
		return err
	}

	// Fail fast if the target exists, so that rerunning a copy that skips
	// existing files doesn't upload every file only to lose the rename. This
	// check alone is racy; createExclusive provides the guarantee.
	_, err = c.stat(ctx, absPath)
	if err == nil {
		return fileAlreadyExistsError{absPath}
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	return c.createExclusive(ctx, absPath, reader)
}

// createExclusive writes a file that must not already exist, and guarantees
// that only one of several concurrent writers creates it. The libs/locker lock
// file relies on this guarantee from filers.
//
// The upload API can't do this: it always replaces the target (observed), and
// per its docs rejects conditional headers such as If-None-Match with 400. The
// rename API can: it returns
// 409 ALREADY_EXISTS if the destination exists, and of several concurrent
// renames onto the same absent destination, exactly one succeeds. That
// atomicity is observed behavior (consistent with renameat(2) with
// RENAME_NOREPLACE), not documented; TestFilerConcurrentExclusiveWrite in
// integration/libs/filer pins it. So upload to a temporary file and rename it
// into place.
func (c *SandboxClient) createExclusive(ctx context.Context, absPath string, reader io.Reader) error {
	tmpPath := exclusiveCreateTempPath(absPath)

	err := c.upload(ctx, tmpPath, reader)
	if err != nil {
		// A failed upload can still have created the file, if the response was
		// lost after the server committed it.
		c.removeTemp(ctx, tmpPath)
		return err
	}

	_, err = c.api.RenamePath(ctx, sandbox.RenamePathRequest{
		Parent:      c.parent,
		Source:      tmpPath,
		Destination: absPath,
	})
	if err != nil {
		// A failed rename leaves the temporary file in place.
		c.removeTemp(ctx, tmpPath)

		// The destination exists. Another writer created it, or it is a
		// directory; os.OpenFile with O_EXCL reports both as fs.ErrExist too.
		if errors.Is(err, apierr.ErrAlreadyExists) {
			return fileAlreadyExistsError{absPath}
		}
		return sandboxPathError(err, absPath)
	}

	return nil
}

// exclusiveCreateTempPath returns the temporary path that createExclusive
// uploads to before renaming it to absPath. It is in the same directory because
// rename fails across filesystems (400), and the directory is the only location
// known to be on the target's filesystem. The random name keeps concurrent
// writers of the same target from sharing one temporary file, and its fixed
// length keeps it within the filesystem's name limit whatever the target's name.
// A temporary file left behind (removeTemp failed or the process was killed) is
// a regular dotfile, visible to listings and recursive copies.
func exclusiveCreateTempPath(absPath string) string {
	return path.Join(path.Dir(absPath), ".databricks-"+rand.Text()+".tmp")
}

// removeTemp deletes a temporary file left by createExclusive. It runs even if
// ctx is cancelled, because cancellation is a common reason the write failed.
// Its error is ignored: a leftover temporary file doesn't change the outcome of
// the write, and reporting it would hide the error that caused the cleanup.
func (c *SandboxClient) removeTemp(ctx context.Context, tmpPath string) {
	_, _ = c.api.DeletePath(context.WithoutCancel(ctx), sandbox.DeletePathRequest{
		Parent: c.parent,
		Path:   tmpPath,
	})
}

// upload creates or replaces the file at absPath, creating missing parents.
func (c *SandboxClient) upload(ctx context.Context, absPath string, reader io.Reader) error {
	err := c.api.UploadFile(ctx, sandbox.UploadFileRequest{
		Parent: c.parent,
		Path:   absPath,
		// The caller owns reader; the SDK must not close it.
		Contents: io.NopCloser(reader),
	})
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
