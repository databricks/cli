package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/databricks/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// TODO: create a mutex for dbfs too
type Mutex interface {
	Lock(ctx context.Context) error
	Unlock(ctx context.Context) error
}

type DeployMutex struct {
	Id              string
	User            string
	AcquisitionTime time.Time
	IsForced        bool
	ProjectRoot     string
}

// Context:
// 1. Test lock and unlock
// 2. create a function to get a new deploy mutex
// 3. make sure ID generated is unique
// 4. then there is the integration test
// 5. stretch: look for opportunities for unit tests
func generateRandomId(length int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getRemoteLock(ctx context.Context, lockFilePath string, apiClient *client.DatabricksClient) (*DeployMutex, error) {
	expectApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/%s",
		strings.TrimLeft(lockFilePath, "/"))

	var res interface{}

	err := apiClient.Get(ctx, expectApiPath, nil, &res)
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
	remoteLock := DeployMutex{}
	err = json.Unmarshal(lockJson, &remoteLock)
	if err != nil {
		return nil, err
	}
	return &remoteLock, nil
}

// TODO: Lets keep a history of all the deployments made.
// 1. When waiting for lock, we can also show that from whom are we waiting to gain a lock from and when they accesses the lock
// 2. For force, delete all the other locks and
func (mu *DeployMutex) Lock(ctx context.Context) error {
	prj := project.Get(ctx)
	wsc := prj.WorkspacesClient()
	lockFilePath := filepath.Join(mu.ProjectRoot, ".bundle/deploy.lock")

	// Do I need to dereference this pointer to marshal ?
	mu.AcquisitionTime = time.Now()
	mutexMetadata, err := json.Marshal(*mu)
	if err != nil {
		return err
	}
	mutexMetadataReader := bytes.NewReader(mutexMetadata)

	// create .bundle dir in remote project root
	err = wsc.Workspace.MkdirsByPath(ctx, filepath.Dir(lockFilePath))
	if err != nil {
		return fmt.Errorf("could not mkdir to put file: %s", err)
	}

	// Try to create the lock file. If we get an error, that means the lock
	// file already exists
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	importApiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=false",
		strings.TrimLeft(lockFilePath, "/"))
	err = apiClient.Post(ctx, importApiPath, mutexMetadataReader, nil)

	if err != nil && strings.Contains(err.Error(), fmt.Sprintf("%s already exists", lockFilePath)) {
		remoteLock, err := getRemoteLock(ctx, lockFilePath, apiClient)
		if err != nil {
			return err
		}
		// TODO: add isForce to message here and convert timestamp to human readable format
		return fmt.Errorf("cannot deploy. %s has been deploying since %v. Use --force to forcibly deploy your bundle", remoteLock.User, remoteLock.AcquisitionTime)
	}
	return nil
}

// TODO: Check you own the lock before deleting it!
func (mu *DeployMutex) Unlock(ctx context.Context) error {
	prj := project.Get(ctx)
	wsc := prj.WorkspacesClient()
	lockFilePath := filepath.Join(mu.ProjectRoot, ".bundle/deploy.lock")
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	remoteLock, err := getRemoteLock(ctx, lockFilePath, apiClient)
	if err != nil {
		return err
	}
	err = wsc.Workspace.Delete(ctx,
		workspace.Delete{
			Path:      lockFilePath,
			Recursive: false,
		},
	)
	return err
}
