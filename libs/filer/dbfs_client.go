package filer

import (
	"cmp"
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/dbfs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/apierr"
	files "github.com/databricks/sdk-go/files/v2"
)

// Type that implements fs.DirEntry for DBFS.
type dbfsDirEntry struct {
	dbfsFileInfo
}

func (entry dbfsDirEntry) Type() fs.FileMode {
	return entry.Mode()
}

func (entry dbfsDirEntry) Info() (fs.FileInfo, error) {
	return entry.dbfsFileInfo, nil
}

// Type that implements fs.FileInfo for DBFS.
type dbfsFileInfo struct {
	fi files.FileInfo
}

func (info dbfsFileInfo) Name() string {
	return path.Base(utils.Deref(info.fi.Path))
}

func (info dbfsFileInfo) Size() int64 {
	return utils.Deref(info.fi.FileSize)
}

func (info dbfsFileInfo) Mode() fs.FileMode {
	mode := fs.ModePerm
	if utils.Deref(info.fi.IsDir) {
		mode |= fs.ModeDir
	}
	return mode
}

func (info dbfsFileInfo) ModTime() time.Time {
	return time.UnixMilli(utils.Deref(info.fi.ModificationTime))
}

func (info dbfsFileInfo) IsDir() bool {
	return utils.Deref(info.fi.IsDir)
}

func (info dbfsFileInfo) Sys() any {
	return info.fi
}

// DbfsClient implements the [Filer] interface for the DBFS backend.
type DbfsClient struct {
	workspaceClient *sdkclients.Clients

	// File operations will be relative to this path.
	root WorkspaceRootPath
}

func NewDbfsClient(w *sdkclients.Clients, root string) (Filer, error) {
	return &DbfsClient{
		workspaceClient: w,

		root: NewWorkspaceRootPath(root),
	}, nil
}

func (w *DbfsClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	overwrite := slices.Contains(mode, OverwriteIfExists)

	// Issue info call before write because it automatically creates parent directories.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	if !slices.Contains(mode, CreateParentDirectories) {
		_, err = w.workspaceClient.Files.GetStatus(ctx, files.GetStatusRequest{Path: new(path.Dir(absPath))})
		if err != nil {
			aerr, ok := errors.AsType[*apierr.APIError](err)
			if !ok {
				return err
			}

			// This API returns a 404 if the file doesn't exist.
			if aerr.HTTPStatusCode() == http.StatusNotFound {
				if apierrs.ErrorCode(aerr) == "RESOURCE_DOES_NOT_EXIST" {
					return noSuchDirectoryError{path.Dir(absPath)}
				}
			}

			return err
		}
	}

	fileMode := dbfs.FileModeWrite
	if overwrite {
		fileMode |= dbfs.FileModeOverwrite
	}
	handle, err := dbfs.Open(ctx, w.workspaceClient.Files, absPath, fileMode)
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return err
		}

		// This API returns a 400 if the file already exists.
		if aerr.HTTPStatusCode() == http.StatusBadRequest {
			if apierrs.ErrorCode(aerr) == "RESOURCE_ALREADY_EXISTS" {
				return fileAlreadyExistsError{absPath}
			}
		}

		return err
	}

	_, err = io.Copy(handle, reader)
	cerr := handle.Close()
	if err == nil {
		err = cerr
	}

	return err
}

func (w *DbfsClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	handle, err := dbfs.Open(ctx, w.workspaceClient.Files, absPath, dbfs.FileModeRead)
	if err != nil {
		// Return error if file is a directory
		if strings.Contains(err.Error(), "cannot open directory for reading") {
			return nil, notAFile{absPath}
		}

		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.HTTPStatusCode() == http.StatusNotFound {
			if apierrs.ErrorCode(aerr) == "RESOURCE_DOES_NOT_EXIST" {
				return nil, fileDoesNotExistError{absPath}
			}
		}

		return nil, err
	}

	// A DBFS handle open for reading does not need to be closed.
	return io.NopCloser(handle), nil
}

func (w *DbfsClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return cannotDeleteRootError{}
	}

	// Issue info call before delete because delete succeeds if the specified path doesn't exist.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	_, err = w.workspaceClient.Files.GetStatus(ctx, files.GetStatusRequest{Path: new(absPath)})
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.HTTPStatusCode() == http.StatusNotFound {
			if apierrs.ErrorCode(aerr) == "RESOURCE_DOES_NOT_EXIST" {
				return fileDoesNotExistError{absPath}
			}
		}

		return err
	}

	recursive := slices.Contains(mode, DeleteRecursively)

	_, err = w.workspaceClient.Files.Delete(ctx, files.DeleteRequest{
		Path:      new(absPath),
		Recursive: new(recursive),
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

	switch aerr.HTTPStatusCode() {
	case http.StatusBadRequest:
		// Anecdotally, this error is returned when attempting to delete a non-empty directory.
		if apierrs.ErrorCode(aerr) == "IO_ERROR" {
			return directoryNotEmptyError{absPath}
		}

		// Since 17th december we are observing the backend return an error_code of BAD_REQUEST
		// instead of IO_ERROR. Doing direct comparison on the error string instead.
		if strings.Contains(aerr.Message(), "Directory is not empty. Use recursive delete or remove contents first.") {
			return directoryNotEmptyError{absPath}
		}
	}

	return err
}

func (w *DbfsClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	res, err := w.workspaceClient.Files.List(ctx, files.ListStatusRequest{Path: new(absPath)})
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.HTTPStatusCode() == http.StatusNotFound {
			if apierrs.ErrorCode(aerr) == "RESOURCE_DOES_NOT_EXIST" {
				return nil, noSuchDirectoryError{absPath}
			}
		}

		return nil, err
	}

	if len(res.Files) == 1 && utils.Deref(res.Files[0].Path) == absPath {
		return nil, notADirectory{absPath}
	}

	info := make([]fs.DirEntry, len(res.Files))
	for i, v := range res.Files {
		info[i] = dbfsDirEntry{fi: v}
	}

	// Sort by name for parity with os.ReadDir.
	slices.SortFunc(info, func(a, b fs.DirEntry) int { return cmp.Compare(a.Name(), b.Name()) })
	return info, nil
}

func (w *DbfsClient) Mkdir(ctx context.Context, name string) error {
	dirPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	_, err = w.workspaceClient.Files.Mkdirs(ctx, files.MkDirsRequest{Path: new(dirPath)})
	return err
}

func (w *DbfsClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	info, err := w.workspaceClient.Files.GetStatus(ctx, files.GetStatusRequest{Path: new(absPath)})
	if err != nil {
		aerr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.HTTPStatusCode() == http.StatusNotFound {
			if apierrs.ErrorCode(aerr) == "RESOURCE_DOES_NOT_EXIST" {
				return nil, fileDoesNotExistError{absPath}
			}
		}

		return nil, err
	}

	return dbfsFileInfo{files.FileInfo{Path: info.Path, IsDir: info.IsDir, FileSize: info.FileSize, ModificationTime: info.ModificationTime}}, nil
}
