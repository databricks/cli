package filer

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/databricks/databricks-sdk-go"
)

// FilesClient implements the [Filer] interface for the Files API backend.
type FilesClient struct {
	workspaceClient *databricks.WorkspaceClient

	// File operations will be relative to this path.
	root RootPath
}

func filesNotImplementedError(fn string) error {
	return fmt.Errorf("filer.%s is not implemented for the Files API", fn)
}

func NewFilesClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	return &FilesClient{
		workspaceClient: w,

		root: NewRootPath(root),
	}, nil
}

func (w *FilesClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	return w.workspaceClient.Files.Upload(ctx, absPath, reader)
}

func (w *FilesClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	absPath, err := w.root.Join(name)
	if err != nil {
		return nil, err
	}

	return w.workspaceClient.Files.Download(ctx, absPath)
}

func (w *FilesClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	return w.workspaceClient.Files.Delete(ctx, absPath)
}

func (w *FilesClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	return nil, filesNotImplementedError("ReadDir")
}

func (w *FilesClient) Mkdir(ctx context.Context, name string) error {
	return filesNotImplementedError("Mkdir")
}

func (w *FilesClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	return nil, filesNotImplementedError("Stat")
}
