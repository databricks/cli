package bundle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/databricks/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/google/uuid"
)

// a mutux on a specified directory in workspace file system.
//
// Only one  DeployLocker can be "active" on a workspace directory. This
// enables exclusive access to the workspace for deployment purposes
//
// users need to acquire this lock before deploying a DAB using
// `bricks bundle deploy`
type DeployLocker struct {
	// unique id for the locker
	Id uuid.UUID
	// creator of this locker
	User string
	// timestamp when this locker became "active" on the the target directory
	AcquisitionTime time.Time
	// forced lockers can override any existing lockers (including other forced ones)
	// on the target directory
	IsForced bool
	// remote root of the project, for which this locker is scoped
	TargetDir string
	// If true, the holder of this locker has exclusive access to target directory
	// to deploy their DAB
	Active bool
}

func GetRemoteLocker(ctx context.Context, wsc *workspaces.WorkspacesClient, lockFilePath string) (*DeployLocker, error) {
	res, err := GetFileContent(ctx, wsc, lockFilePath)
	if err != nil {
		return nil, err
	}
	lockJson, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	remoteLock := DeployLocker{}
	err = json.Unmarshal(lockJson, &remoteLock)
	if err != nil {
		return nil, err
	}
	return &remoteLock, nil
}

// idempotent
func (locker *DeployLocker) safePutFile(ctx context.Context, wsc *workspaces.WorkspacesClient, path string, content []byte) error {
	contentReader := bytes.NewReader(content)
	// TODO: Consider reading the remote locker file to ensure we hold the lock
	// This hedges against race conditions during forced deployment
	if !locker.Active {
		return fmt.Errorf("failed to put file. Lock not held to safely mutate workspace files")
	}
	// workspace mkdirs is idempotent
	err := wsc.Workspace.MkdirsByPath(ctx, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("could not mkdir to put file: %s", err)
	}
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	apiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=true",
		strings.TrimLeft(path, "/"))
	return apiClient.Post(ctx, apiPath, contentReader, nil)
}

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
func postFile(ctx context.Context, wsc *workspaces.WorkspacesClient, path string, content []byte) error {
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

func (locker *DeployLocker) postLockFile(ctx context.Context, wsc *workspaces.WorkspacesClient) error {
	locker.AcquisitionTime = time.Now()
	lockerContent, err := json.Marshal(*locker)
	if err != nil {
		return err
	}
	return postFile(ctx, wsc, locker.RemotePath(), lockerContent)
}

func (locker *DeployLocker) Lock(ctx context.Context, wsc *workspaces.WorkspacesClient) error {
	if locker.Active {
		return fmt.Errorf("locker already active")
	}
	err := locker.postLockFile(ctx, wsc)
	if err != nil && strings.Contains(err.Error(), fmt.Sprintf("%s already exists", locker.RemotePath())) {
		remoteLocker, err := GetRemoteLocker(ctx, wsc, locker.RemotePath())
		if err != nil {
			return fmt.Errorf("failed to get remote lock file: %s", err)
		}
		// TODO: convert timestamp to human readable format
		if remoteLocker.IsForced {
			return fmt.Errorf("ongoing forced deployment by %s since %v. Use --force to override current forced deployment", remoteLocker.User, remoteLocker.AcquisitionTime)
		} else {
			return fmt.Errorf("ongoing deployment by %s since %v. Use --force to forcibly deploy your bundle", remoteLocker.User, remoteLocker.AcquisitionTime)
		}
	}
	locker.Active = true
	return nil
}

func (locker *DeployLocker) RemotePath() string {
	return filepath.Join(locker.TargetDir, ".bundle/deploy.lock")
}

func CreateLocker(user string, isForced bool, targetDir string) (*DeployLocker, error) {
	return &DeployLocker{
		Id:        uuid.New(),
		IsForced:  isForced,
		TargetDir: targetDir,
		User:      user,
		Active:    false,
	}, nil
}

func (locker *DeployLocker) Unlock(ctx context.Context, wsc *workspaces.WorkspacesClient) error {
	if !locker.Active {
		return fmt.Errorf("only active lockers can be unlocked")
	}

	remoteLocker, err := GetRemoteLocker(ctx, wsc, locker.RemotePath())
	if err != nil {
		return err
	}
	if remoteLocker.Id == locker.Id {
		err = wsc.Workspace.Delete(ctx,
			workspace.Delete{
				Path:      locker.RemotePath(),
				Recursive: false,
			},
		)
		locker.Active = false
	} else {
		err = fmt.Errorf("this deployment does not hold lock on workspace project dir. Current active deployment lock was acquired by %s at %v", remoteLocker.User, remoteLocker.AcquisitionTime)
		locker.Active = false
	}
	return err
}
