package utilities

import (
	"bytes"
	"context"
	"encoding/json"
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
// Get the file contents of a json file in workspace
// TODO(Nov 2022): add method in go sdk to get the raw bytes from response of an API
//
// TODO(Nov 2022): talk to eng-files team about what the response structure would look like.
//       This function would have to be modfified probably in the future once this
//       API goes to public preview
func GetRawJsonFileContent(ctx context.Context, wsc *databricks.WorkspaceClient, path string) ([]byte, error) {
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return nil, err
	}
	exportApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(path, "/"))

	var res json.RawMessage

	err = apiClient.Do(ctx, http.MethodGet, exportApiPath, nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file %s: %s", path, err)
	}
	return res, nil
}

func WriteFile(ctx context.Context, wsc *databricks.WorkspaceClient, path string, content []byte, overwrite bool) error {
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

	return apiClient.Do(ctx, http.MethodPost, importApiPath, bytes.NewReader(content), nil)
}
