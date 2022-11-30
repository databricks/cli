package utilities

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)


// NOTE: This API is only available for files in /Repos if a workspace has repos
// in workspace enabled and files in workspace not enabled
//
// Right now the GET workspace-file api returns the raw file content as the
// reponse body which then the go-sdk unmarshals in apiClient.Do
//
// The consequences of this?
// 1. Using this function on workspace files that are not json formatted will error out
// 2. The expected runtime type of the returned result is map[string]interface{}
func GetJsonFileContent(ctx context.Context, wsc *databricks.WorkspaceClient, path string) (interface{}, error) {
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return nil, err
	}
	exportApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(path, "/"))

	var res interface{}

	err = apiClient.Do(ctx, http.MethodGet, exportApiPath, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file %s: %s", path, err)
	}
	return res, nil
}

// not idempotent. errors out if file exists
func WriteFile(ctx context.Context, wsc *databricks.WorkspaceClient, path string, content []byte, overwrite bool) error {
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
		"/api/2.0/workspace-files/import-file/%s?overwrite=%s",
		strings.TrimLeft(path, "/"), strconv.FormatBool(overwrite))

	return apiClient.Do(ctx, http.MethodPost, importApiPath, contentReader, nil)
}
