package filer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/exp/slices"
)

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
		strings.TrimLeft(absPath, "/"),
		overwrite,
	)

	// Buffer the file contents because we may need to retry below and we cannot read twice.
	body, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	err = w.apiClient.Do(ctx, http.MethodPost, urlPath, body, nil)

	// If we got an API error we deal with it below.
	aerr, ok := err.(apierr.APIError)
	if !ok {
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
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(res), nil
}

func (w *WorkspaceFilesClient) Delete(ctx context.Context, name string) error {
	absPath, err := w.root.Join(name)
	if err != nil {
		return err
	}

	return w.workspaceClient.Workspace.Delete(ctx, workspace.Delete{
		Path:      absPath,
		Recursive: false,
	})
}
