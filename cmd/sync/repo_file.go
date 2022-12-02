package sync

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

type repoFiles struct {
	repoRoot  string
	localRoot string
	wsc       *databricks.WorkspaceClient
}

func CreateRepoFiles(repoRoot, localRoot string, wsc *databricks.WorkspaceClient) *repoFiles {
	return &repoFiles{
		repoRoot:  repoRoot,
		localRoot: localRoot,
		wsc:       wsc,
	}
}

// TODO: error out on symlinks
// TODO add tests for bad relative paths and symlinks
func (r *repoFiles) getCleanRemotePath(relativePath string) (string, error) {
	if strings.Contains(relativePath, `..`) {
		return "", fmt.Errorf(`file relative path %s contains forbidden pattern ".."`, relativePath)
	}
	cleanRelativePath := path.Clean(relativePath)
	localPath := path.Join(r.repoRoot, cleanRelativePath)
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	// we ignore sym links
	if fileInfo.Mode() == os.ModeSymlink {

	}
}

func (r *repoFiles) putFile(ctx context.Context, relativePath string) error {
	content, err := os.ReadFile(filepath.Join(r.localRoot, relativePath))
	if err != nil {
		return err
	}
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	apiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=true",
		strings.TrimLeft(remotePath, "/"))
}
