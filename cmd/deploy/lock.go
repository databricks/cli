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

// TODO: create a mutex for dbfs too
type Mutex interface {
	Lock(ctx context.Context) error
	Unlock(ctx context.Context) error
}

type DeployLocker struct {
	Id              uuid.UUID
	User            string
	AcquisitionTime time.Time
	IsForced        bool
	TargetDir       string
}

// Context:
// 1. Test lock and unlock
// 2. create a function to get a new deploy mutex
// 3. make sure ID generated is unique
// 4. then there is the integration test
// 5. stretch: look for opportunities for unit tests
// func generateRandomId(length int) string {
// 	var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
// 	rand.Seed(time.Now().UnixNano())
// 	b := make([]rune, length)
// 	for i := range b {
// 		b[i] = letters[rand.Intn(len(letters))]
// 	}
// 	return string(b)
// }

func getRemoteLocker(ctx context.Context, lockFilePath string) (*DeployLocker, error) {
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
	// TODO: handle the race condition when the file gets deleted before you can read
	// it.
	// NOTE: https://databricks.atlassian.net/browse/ES-510449
	// Do some error parsing maybe, add on previos error for more context and
	// add on suggestion for the user to retry deployment
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

func postFile(ctx context.Context, path string, content []byte) error {
	contentReader := bytes.NewReader(content)
	wsc := project.Get(ctx).WorkspacesClient()
	apiClient, err := client.New(wsc.Config)
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

// Test what happens if two simulataenous requests to create a file go out

func (locker *DeployLocker) Lock(ctx context.Context) error {
	err := locker.postLockFile(ctx)
	if err != nil && strings.Contains(err.Error(), fmt.Sprintf("%s already exists", locker.remotePath())) {
		remoteLocker, err := getRemoteLocker(ctx, locker.remotePath())
		if err != nil {
			return err
		}
		// TODO: add isForce to message here and convert timestamp to human readable format
		return fmt.Errorf("cannot deploy. %s has been deploying since %v. Use --force to forcibly deploy your bundle", remoteLocker.User, remoteLocker.AcquisitionTime)
	}
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
	}, nil
}

// TODO: Check you own the lock before deleting it!
func (locker *DeployLocker) Unlock(ctx context.Context) error {
	wsc := project.Get(ctx).WorkspacesClient()
	remoteLocker, err := getRemoteLocker(ctx, locker.remotePath())
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
	} else {
		err = fmt.Errorf("tried to unlock unacquired locker: %+v. Current owner locker of target: %+v. ", locker, remoteLocker)
	}
	return err
}
