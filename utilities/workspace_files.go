package utilities

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go/databricks/client"
	"github.com/databricks/databricks-sdk-go/workspaces"
)

func GetFileContent(ctx context.Context, wsc *workspaces.WorkspacesClient, path string) (interface{}, error) {
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
	err = apiClient.Get(ctx, exportApiPath, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file %s: %s", path, err)
	}
	return res, nil
}

// not idempotent. errors out if file exists
func PostFile(ctx context.Context, wsc *workspaces.WorkspacesClient, path string, content []byte) error {
	contentReader := bytes.NewReader(content)
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	err = wsc.Workspace.MkdirsByPath(ctx, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("could not mkdir to put file: %s", err)
	}
	if err != nil {
		return err
	}
	importApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=false",
		strings.TrimLeft(path, "/"))
	return apiClient.Post(ctx, importApiPath, contentReader, nil)
}
