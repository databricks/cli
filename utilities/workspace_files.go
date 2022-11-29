package utilities

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// TODO: This returns
// 1. an error if the file contents are not json
// 2. an map[string]interface{} if the contents are json
//
// Make changes to read workspace files whose content body is not json
func GetFile(ctx context.Context, wsc *databricks.WorkspaceClient, path string) (interface{}, error) {
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return nil, err
	}
	exportApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(path, "/"))

	var res interface{}

	// NOTE: azure workspaces return misleading messages when a file does not exist
	// see: https://databricks.atlassian.net/browse/ES-510449
	err = apiClient.Do(ctx, http.MethodGet, exportApiPath, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file %s: %s", path, err)
	}
	return res, nil
}

// not idempotent. errors out if file exists
func PostFile(ctx context.Context, wsc *databricks.WorkspaceClient, path string, content []byte) error {
	contentReader := bytes.NewReader(content)
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	err = wsc.Workspace.MkdirsByPath(ctx, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("could not mkdir to post file: %s", err)
	}
	importApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=false",
		strings.TrimLeft(path, "/"))
	return apiClient.Do(ctx, http.MethodPost, importApiPath, contentReader, nil)
}
