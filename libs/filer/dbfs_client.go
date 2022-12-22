package filer

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dbfs"
	"golang.org/x/exp/slices"
)

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

	fileMode := dbfs.FileModeWrite
	if slices.Contains(mode, OverwriteIfExists) {
		fileMode |= dbfs.FileModeOverwrite
	}
	handle, err := w.workspaceClient.Dbfs.Open(ctx, absPath, fileMode)
	if err != nil {
		var aerr apierr.APIError
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

	handle, err := w.workspaceClient.Dbfs.Open(ctx, absPath, dbfs.FileModeRead)
	if err != nil {
		var aerr apierr.APIError
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

	return handle, nil
}

func (w *DbfsClient) Delete(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	// Issue info call before delete because delete succeeds if the specified path doesn't exist.
	//
	// For discussion: we could decide this is actually convenient, remove the call below,
	// and apply the same semantics for the WSFS filer.
	//
	_, err = w.workspaceClient.Dbfs.GetStatusByPath(ctx, absPath)
	if err != nil {
		var aerr apierr.APIError
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

	return w.workspaceClient.Dbfs.Delete(ctx, dbfs.Delete{
		Path:      absPath,
		Recursive: false,
	})
}
