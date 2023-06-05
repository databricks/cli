package repofiles

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type RepoFileOptions struct {
	OverwriteIfExists bool
}

// RepoFiles wraps reading and writing into a remote repo with safeguards to prevent
// accidental deletion of repos and more robust methods to overwrite workspac e files
type RepoFiles struct {
	*RepoFileOptions

	repoRoot        string
	localRoot       string
	workspaceClient *databricks.WorkspaceClient
	f               filer.Filer
}

func Create(repoRoot, localRoot string, w *databricks.WorkspaceClient, opts *RepoFileOptions) (*RepoFiles, error) {
	// override default timeout to support uploading larger files
	w.Config.HTTPTimeoutSeconds = 600

	// create filer to interact with WSFS
	f, err := filer.NewWorkspaceFilesClient(w, repoRoot)
	if err != nil {
		return nil, err
	}
	return &RepoFiles{
		repoRoot:        repoRoot,
		localRoot:       localRoot,
		workspaceClient: w,
		RepoFileOptions: opts,
		f:               f,
	}, nil
}

func (r *RepoFiles) remotePath(relativePath string) (string, error) {
	fullPath := path.Join(r.repoRoot, relativePath)
	cleanFullPath := path.Clean(fullPath)
	if !strings.HasPrefix(cleanFullPath, r.repoRoot) {
		return "", fmt.Errorf("relative file path is not inside repo root: %s", relativePath)
	}
	// path.Clean will remove any trailing / so it's enough to check cleanFullPath == r.repoRoot
	if cleanFullPath == r.repoRoot {
		return "", fmt.Errorf("file path relative to repo root cannot be empty: %s", relativePath)
	}
	return cleanFullPath, nil
}

func (r *RepoFiles) readLocal(relativePath string) ([]byte, error) {
	localPath := filepath.Join(r.localRoot, relativePath)
	return os.ReadFile(localPath)
}

func (r *RepoFiles) writeRemote(ctx context.Context, relativePath string, content []byte) error {
	if !r.OverwriteIfExists {
		return r.f.Write(ctx, relativePath, bytes.NewReader(content), filer.CreateParentDirectories)
	}

	err := r.f.Write(ctx, relativePath, bytes.NewReader(content), filer.CreateParentDirectories, filer.OverwriteIfExists)

	// TODO(pietern): Use the new FS interface to avoid needing to make a recursive
	// delete call here. This call is dangerous
	if err != nil {
		// Delete any artifact files incase non overwriteable by the current file
		// type and thus are failing the PUT request.
		// files, folders and notebooks might not have been cleaned up and they
		// can't overwrite each other. If a folder `foo` exists, then attempts to
		// PUT a file `foo` will fail
		remotePath, err := r.remotePath(relativePath)
		if err != nil {
			return err
		}
		err = r.workspaceClient.Workspace.Delete(ctx,
			workspace.Delete{
				Path:      remotePath,
				Recursive: true,
			},
		)
		// ignore RESOURCE_DOES_NOT_EXIST here incase nothing existed at remotePath
		var aerr *apierr.APIError
		if errors.As(err, &aerr) && aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
			err = nil
		}
		if err != nil {
			return err
		}

		// Attempt to write the file again, this time without the CreateParentDirectories and
		// OverwriteIfExists flags
		return r.f.Write(ctx, relativePath, bytes.NewReader(content))
	}
	return nil
}

func (r *RepoFiles) deleteRemote(ctx context.Context, relativePath string) error {
	return r.f.Delete(ctx, relativePath)
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
	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.ErrorCode == "RESOURCE_DOES_NOT_EXIST" {
		err = nil
	}
	return nil
}
