package filer

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/files"
	"golang.org/x/exp/slices"
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
	return path.Base(info.fi.Path)
}

func (info dbfsFileInfo) Size() int64 {
	return info.fi.FileSize
}

func (info dbfsFileInfo) Mode() fs.FileMode {
	mode := fs.ModePerm
	if info.fi.IsDir {
		mode |= fs.ModeDir
	}
	return mode
}

func (info dbfsFileInfo) ModTime() time.Time {
	return time.UnixMilli(info.fi.ModificationTime)
}

func (info dbfsFileInfo) IsDir() bool {
	return info.fi.IsDir
}

func (info dbfsFileInfo) Sys() any {
	return info.fi
}

// DbfsClient implements the [Filer] interface for the DBFS backend.
type DbfsClient struct {
	workspaceClient *databricks.WorkspaceClient

	// File operations will be relative to this path.
	root RootPath
}

func NewDbfsClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	return &DbfsClient{
		workspaceClient: w,

		root: NewRootPath(root),
	}, nil
}

func (w *DbfsClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	fileMode := files.FileModeWrite
	if slices.Contains(mode, OverwriteIfExists) {
		fileMode |= files.FileModeOverwrite
	}

	// Issue info call before write because it automatically creates parent directories.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	if !slices.Contains(mode, CreateParentDirectories) {
		_, err = w.workspaceClient.Dbfs.GetStatusByPath(ctx, path.Dir(absPath))
		if err != nil {
			var aerr *apierr.APIError
			if !errors.As(err, &aerr) {
				return err
			}

			// This API returns a 404 if the file doesn't exist.
			if aerr.StatusCode == http.StatusNotFound {
				if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
					return NoSuchDirectoryError{path.Dir(absPath)}
				}
			}

			return err
		}
	}

	handle, err := w.workspaceClient.Dbfs.Open(ctx, absPath, fileMode)
	if err != nil {
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return err
		}

		// This API returns a 400 if the file already exists.
		if aerr.StatusCode == http.StatusBadRequest {
			if aerr.ErrorCode == "RESOURCE_ALREADY_EXISTS" {
				return FileAlreadyExistsError{absPath}
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

func (w *DbfsClient) Read(ctx context.Context, name string) (io.Reader, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	// This stat call serves two purposes:
	// 1. Checks file at path exists, and throws an error if it does not
	// 2. Allows use to error out if the path is a directory. This is needed
	// because the Dbfs.Open method on the SDK does not error when the path is
	// a directory
	// TODO: remove this stat call on go sdk bump. https://github.com/databricks/cli/issues/450
	stat, err := w.Stat(ctx, name)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, NotAFile{absPath}
	}

	return w.workspaceClient.Dbfs.Open(ctx, absPath, files.FileModeRead)
}

func (w *DbfsClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Illegal to delete the root path.
	if absPath == w.root.rootPath {
		return CannotDeleteRootError{}
	}

	// Issue info call before delete because delete succeeds if the specified path doesn't exist.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	_, err = w.workspaceClient.Dbfs.GetStatusByPath(ctx, absPath)
	if err != nil {
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return FileDoesNotExistError{absPath}
			}
		}

		return err
	}

	recursive := false
	if slices.Contains(mode, DeleteRecursively) {
		recursive = true
	}

	err = w.workspaceClient.Dbfs.Delete(ctx, files.Delete{
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
		// Anecdotally, this error is returned when attempting to delete a non-empty directory.
		if aerr.ErrorCode == "IO_ERROR" {
			return DirectoryNotEmptyError{absPath}
		}
	}

	return err
}

func (w *DbfsClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	res, err := w.workspaceClient.Dbfs.ListByPath(ctx, absPath)
	if err != nil {
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, NoSuchDirectoryError{absPath}
			}
		}

		return nil, err
	}

	if len(res.Files) == 1 && res.Files[0].Path == absPath {
		return nil, NotADirectory{absPath}
	}

	info := make([]fs.DirEntry, len(res.Files))
	for i, v := range res.Files {
		info[i] = dbfsDirEntry{dbfsFileInfo: dbfsFileInfo{fi: v}}
	}

	// Sort by name for parity with os.ReadDir.
	sort.Slice(info, func(i, j int) bool { return info[i].Name() < info[j].Name() })
	return info, nil
}

func (w *DbfsClient) Mkdir(ctx context.Context, name string) error {
	dirPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	return w.workspaceClient.Dbfs.MkdirsByPath(ctx, dirPath)
}

func (w *DbfsClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	info, err := w.workspaceClient.Dbfs.GetStatusByPath(ctx, absPath)
	if err != nil {
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the file doesn't exist.
		if aerr.StatusCode == http.StatusNotFound {
			if aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
				return nil, FileDoesNotExistError{absPath}
			}
		}

		return nil, err
	}

	return dbfsFileInfo{*info}, nil
}
