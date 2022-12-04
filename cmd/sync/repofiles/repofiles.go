package repofiles

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// TODO: add a comment about this class, how it only sanitizes relative paths and
// has no checks for repo roots
//
// Should we add these checks?
type RepoFiles struct {
	repoRoot        string
	localRoot       string
	workspaceClient *databricks.WorkspaceClient
}

func Create(repoRoot, localRoot string, workspaceClient *databricks.WorkspaceClient) *RepoFiles {
	return &RepoFiles{
		repoRoot:        repoRoot,
		localRoot:       localRoot,
		workspaceClient: workspaceClient,
	}
}

// TODO add tests for bad relative paths and symlinks
func cleanPath(relativePath string) (string, error) {
	cleanRelativePath := path.Clean(relativePath)
	if strings.Contains(cleanRelativePath, `..`) {
		return "", fmt.Errorf(`file relative path %s contains forbidden pattern ".."`, relativePath)
	}
	if cleanRelativePath == "" || cleanRelativePath == "/" || cleanRelativePath == "." {
		return "", fmt.Errorf("file path relative to repo root cannot be empty: %s", relativePath)
	}
	return cleanRelativePath, nil
}

func (r *RepoFiles) remotePath(relativePath string) (string, error) {
	cleanRelativePath, err := cleanPath(relativePath)
	if err != nil {
		return "", err
	}
	return path.Join(r.repoRoot, cleanRelativePath), nil
}

func (r *RepoFiles) localPath(relativePath string) (string, error) {
	cleanRelativePath, err := cleanPath(relativePath)
	if err != nil {
		return "", err
	}
	return filepath.Join(r.localRoot, cleanRelativePath), nil
}

func (r *RepoFiles) readLocal(relativePath string) ([]byte, error) {
	localPath, err := r.localPath(relativePath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(localPath)
}

func (r *RepoFiles) writeRemote(ctx context.Context, relativePath string, content []byte) error {
	apiClient, err := client.New(r.workspaceClient.Config)
	if err != nil {
		return err
	}
	remotePath, err := r.remotePath(relativePath)
	if err != nil {
		return err
	}
	apiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=true",
		strings.TrimLeft(remotePath, "/"))

	// TODO: This might fail if the content is not io.reader (and bytes instead)
	// test and if it fails then change this
	err = apiClient.Do(ctx, http.MethodPost, apiPath, content, nil)

	// TODO: check if the error returned here is generic or mentions that directory
	// creation failed
	// Attempt file creation again this time also creating intermidiate dirs
	// incase they were missing
	if err != nil {
		err := r.workspaceClient.Workspace.MkdirsByPath(ctx, path.Dir(remotePath))
		if err != nil {
			return fmt.Errorf("could not mkdir to put file: %s", err)
		}
		err = apiClient.Do(ctx, http.MethodPost, apiPath, content, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RepoFiles) deleteRemote(ctx context.Context, relativePath string) error {
	remotePath, err := r.remotePath(relativePath)
	if err != nil {
		return err
	}
	return r.workspaceClient.Workspace.Delete(ctx,
		workspace.Delete{
			Path:      remotePath,
			Recursive: false,
		},
	)
}

// The API calls for a python script foo.py would be
// `PUT foo.py`
// `DELETE foo.py`
//
// The API calls for a python notebook foo.py would be
// `PUT foo.py`
// `DELETE foo`
//
// The workspace file system backend strips .py from the file name if the python
// file is a notebook
func (r *RepoFiles) PutFile(ctx context.Context, relativePath string) error {
	content, err := r.readLocal(relativePath)
	if err != nil {
		return err
	}

	return r.writeRemote(ctx, relativePath, content)
}

func (r *RepoFiles) DeleteFile(ctx context.Context, relativePath string) error {
	err := r.deleteRemote(ctx, relativePath)

	// We explictly ignore RESOURCE_DOES_NOT_EXIST error to make delete idempotent
	if val, ok := err.(apierr.APIError); ok && val.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
		return nil
	}
	return nil
}
