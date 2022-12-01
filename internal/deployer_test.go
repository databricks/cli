package internal

// import (
// 	"context"
// 	"fmt"
// 	"math/rand"
// 	"os"
// 	"path/filepath"
// 	"sync"
// 	"testing"
// 	"time"

// 	"github.com/databricks/bricks/bundle/deployer"
// 	"github.com/databricks/databricks-sdk-go"
// 	"github.com/hashicorp/go-version"
// 	"github.com/hashicorp/hc-install/product"
// 	"github.com/hashicorp/hc-install/releases"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func setupTerraformBinary(t *testing.T) (string, error) {
// 	installer := releases.ExactVersion{
// 		Product: product.Terraform,
// 		Version: version.Must(version.NewVersion("1.2.4")),
// 	}
// 	execPath, err := installer.Install(context.TODO())
// 	if err != nil {
// 		return "", err
// 	}
// 	t.Cleanup(func() {
// 		installer.Remove(context.TODO())
// 	})
// 	return execPath, nil
// }

// func createLocalTestProjectWithTfConfig(t *testing.T, id string) {
// 	localRoot := createLocalTestProject(t)
// 	err := os.MkdirAll(filepath.Join(localRoot, ".databricks/bundle"), os.ModeDir)
// 	require.NoError(t, err)
// 	os.WriteFile(filepath.Join(localRoot, ".databricks/bundle/main.tf"), []byte(
// 		fmt.Sprintf(`

// 		`)
// 	), os.ModePerm)
// }

// func TestAccDeploy(t *testing.T) {
// 	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))
// 	ctx := context.TODO()
// 	wsc := databricks.Must(databricks.NewWorkspaceClient())
// 	localProjectRoot := createLocalTestProject(t)
// 	remoteProjectRoot := createRemoteTestProject(t, "deploy-acc-", wsc)

// 	tfExecPath, err := setupTerraformBinary(t)
// 	assert.NoError(t, err)

// 	numConcurrentDeployers := 4
// 	deployErrs := make([]error, numConcurrentDeployers)
// 	deployers := make([]*deployer.Deployer, numConcurrentDeployers)

// 	for i := 0; i < numConcurrentDeployers; i++ {
// 		deployers[i], err = deployer.Create(ctx, "development", localProjectRoot, remoteProjectRoot, wsc)
// 		require.NoError(t, err)
// 	}

// 	var wg sync.WaitGroup
// 	for i := 0; i < numConcurrentDeployers; i++ {
// 		wg.Add(1)
// 		currentIndex := i
// 		go func() {
// 			defer wg.Done()
// 			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
// 			deployErrs[currentIndex] = deployers[currentIndex].Lock(ctx)
// 		}()
// 	}
// 	wg.Wait()
// }
