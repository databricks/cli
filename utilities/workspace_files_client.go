package utilities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"golang.org/x/exp/slices"
)

// WorkspaceFilesClient implements the Files-in-Workspace API.
type WorkspaceFilesClient struct {
	workspaceClient *databricks.WorkspaceClient
	apiClient       *client.DatabricksClient

	// File operations will be relative to this path.
	root string
}

func NewWorkspaceFilesClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}

	return &WorkspaceFilesClient{
		workspaceClient: w,
		apiClient:       apiClient,

		root: root,
	}, nil
}

func (w *WorkspaceFilesClient) absName(name string) (string, error) {
	absName := path.Join(w.root, name)

	// Don't allow escaping the specified root using relative paths.
	if !strings.HasPrefix(absName, w.root) {
		return "", fmt.Errorf("invalid path: %s", name)
	}

	return absName, nil
}

func (w *WorkspaceFilesClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	absName, err := w.absName(name)
	if err != nil {
		return err
	}

	// Remove leading "/" so we can use it in the URL.
	overwrite := slices.Contains(mode, OverwriteIfExists)
	urlPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=%t",
		strings.TrimLeft(absName, "/"),
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
			return NoSuchDirectoryError{path.Dir(absName)}
		}

		// Create parent directory.
		err = w.workspaceClient.Workspace.MkdirsByPath(ctx, path.Dir(absName))
		if err != nil {
			return fmt.Errorf("could not mkdir to post file: %s", err)
		}

		// Retry without CreateParentDirectories mode flag.
		return w.Write(ctx, name, bytes.NewReader(body))
	}

	// This API returns 409 if the file already exists.
	if aerr.StatusCode == http.StatusConflict {
		return FileAlreadyExistsError{absName}
	}

	return err
}

func (w *WorkspaceFilesClient) Read(ctx context.Context, name string) (io.Reader, error) {
	absName, err := w.absName(name)
	if err != nil {
		return nil, err
	}

	// Remove leading "/" so we can use it in the URL.
	urlPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(absName, "/"),
	)

	// Update to []byte after https://github.com/databricks/databricks-sdk-go/pull/247 is merged.
	var res json.RawMessage
	err = w.apiClient.Do(ctx, http.MethodGet, urlPath, nil, &res)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(res), nil
}
