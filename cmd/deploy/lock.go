package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/databricks/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
)

type DeployLocker struct {
	Id              uuid.UUID
	User            string
	AcquisitionTime time.Time
	IsForced        bool
	TargetDir       string
	Active          bool
}

func GetRemoteLocker(ctx context.Context, lockFilePath string) (*DeployLocker, error) {
	wsc := project.Get(ctx).WorkspacesClient()
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return nil, err
	}
	expectApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(lockFilePath, "/"))

	var res interface{}

	err = apiClient.Get(ctx, expectApiPath, nil, &res)

	// NOTE: azure workspaces return misleading messages when a file does not exist
	// see: https://databricks.atlassian.net/browse/ES-510449
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote deployment lock file: %s", err)
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

func postFile(ctx context.Context, path string, content []byte) error {
	contentReader := bytes.NewReader(content)
	wsc := project.Get(ctx).WorkspacesClient()
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

func (locker *DeployLocker) postLockFile(ctx context.Context) error {
	locker.AcquisitionTime = time.Now()
	lockerContent, err := json.Marshal(*locker)
	if err != nil {
		return err
	}
	return postFile(ctx, locker.remotePath(), lockerContent)
}

func (locker *DeployLocker) Lock(ctx context.Context) error {
	if locker.Active {
		return fmt.Errorf("locker already active")
	}
	err := locker.postLockFile(ctx)
	if err != nil && strings.Contains(err.Error(), fmt.Sprintf("%s already exists", locker.remotePath())) {
		remoteLocker, err := GetRemoteLocker(ctx, locker.remotePath())
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

func (locker *DeployLocker) remotePath() string {
	return filepath.Join(locker.TargetDir, ".bundle/deploy.lock")
}

func CreateLocker(ctx context.Context, isForced bool, targetDir string) (*DeployLocker, error) {
	prj := project.Get(ctx)
	user, err := prj.Me()
	if err != nil {
		return nil, err
	}

	return &DeployLocker{
		Id:        uuid.New(),
		IsForced:  isForced,
		TargetDir: targetDir,
		User:      user.UserName,
		Active:    false,
	}, nil
}

func (locker *DeployLocker) Unlock(ctx context.Context) error {
	if !locker.Active {
		return fmt.Errorf("only active lockers can be unlocked")
	}

	wsc := project.Get(ctx).WorkspacesClient()
	remoteLocker, err := GetRemoteLocker(ctx, locker.remotePath())
	if err != nil {
		return err
	}
	if remoteLocker.Id == locker.Id {
		err = wsc.Workspace.Delete(ctx,
			workspace.Delete{
				Path:      locker.remotePath(),
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
