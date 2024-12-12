package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/internal/testutil"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

func GenerateNotebookTasks(notebookPath string, versions []string, nodeTypeId string) []jobs.SubmitTask {
	tasks := make([]jobs.SubmitTask, 0)
	for i := 0; i < len(versions); i++ {
		task := jobs.SubmitTask{
			TaskKey: fmt.Sprintf("notebook_%s", strings.ReplaceAll(versions[i], ".", "_")),
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: notebookPath,
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     versions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func GenerateSparkPythonTasks(notebookPath string, versions []string, nodeTypeId string) []jobs.SubmitTask {
	tasks := make([]jobs.SubmitTask, 0)
	for i := 0; i < len(versions); i++ {
		task := jobs.SubmitTask{
			TaskKey: fmt.Sprintf("spark_%s", strings.ReplaceAll(versions[i], ".", "_")),
			SparkPythonTask: &jobs.SparkPythonTask{
				PythonFile: notebookPath,
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     versions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func GenerateWheelTasks(wheelPath string, versions []string, nodeTypeId string) []jobs.SubmitTask {
	tasks := make([]jobs.SubmitTask, 0)
	for i := 0; i < len(versions); i++ {
		task := jobs.SubmitTask{
			TaskKey: fmt.Sprintf("whl_%s", strings.ReplaceAll(versions[i], ".", "_")),
			PythonWheelTask: &jobs.PythonWheelTask{
				PackageName: "my_test_code",
				EntryPoint:  "run",
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     versions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
			Libraries: []compute.Library{
				{Whl: wheelPath},
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func TemporaryWorkspaceDir(t testutil.TestingT, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	basePath := fmt.Sprintf("/Users/%s/%s", me.UserName, testutil.RandomName("integration-test-wsfs-"))

	t.Logf("Creating %s", basePath)
	err = w.Workspace.MkdirsByPath(ctx, basePath)
	require.NoError(t, err)

	// Remove test directory on test completion.
	t.Cleanup(func() {
		t.Logf("Removing %s", basePath)
		err := w.Workspace.Delete(ctx, workspace.Delete{
			Path:      basePath,
			Recursive: true,
		})
		if err == nil || apierr.IsMissing(err) {
			return
		}
		t.Logf("Unable to remove temporary workspace directory %s: %#v", basePath, err)
	})

	return basePath
}

func TemporaryDbfsDir(t testutil.TestingT, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	path := fmt.Sprintf("/tmp/%s", testutil.RandomName("integration-test-dbfs-"))

	t.Logf("Creating DBFS folder:%s", path)
	err := w.Dbfs.MkdirsByPath(ctx, path)
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Logf("Removing DBFS folder:%s", path)
		err := w.Dbfs.Delete(ctx, files.Delete{
			Path:      path,
			Recursive: true,
		})
		if err == nil || apierr.IsMissing(err) {
			return
		}
		t.Logf("unable to remove temporary dbfs directory %s: %#v", path, err)
	})

	return path
}

// Create a new UC volume in a catalog called "main" in the workspace.
func TemporaryUcVolume(t testutil.TestingT, w *databricks.WorkspaceClient) string {
	ctx := context.Background()

	// Create a schema
	schema, err := w.Schemas.Create(ctx, catalog.CreateSchema{
		CatalogName: "main",
		Name:        testutil.RandomName("test-schema-"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := w.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{
			FullName: schema.FullName,
		})
		require.NoError(t, err)
	})

	// Create a volume
	volume, err := w.Volumes.Create(ctx, catalog.CreateVolumeRequestContent{
		CatalogName: "main",
		SchemaName:  schema.Name,
		Name:        "my-volume",
		VolumeType:  catalog.VolumeTypeManaged,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := w.Volumes.Delete(ctx, catalog.DeleteVolumeRequest{
			Name: volume.FullName,
		})
		require.NoError(t, err)
	})

	return path.Join("/Volumes", "main", schema.Name, volume.Name)
}

func TemporaryRepo(t testutil.TestingT, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, testutil.RandomName("integration-test-repo-"))

	t.Logf("Creating repo:%s", repoPath)
	repoInfo, err := w.Repos.Create(ctx, workspace.CreateRepoRequest{
		Url:      "https://github.com/databricks/cli",
		Provider: "github",
		Path:     repoPath,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Logf("Removing repo: %s", repoPath)
		err := w.Repos.Delete(ctx, workspace.DeleteRepoRequest{
			RepoId: repoInfo.Id,
		})
		if err == nil || apierr.IsMissing(err) {
			return
		}
		t.Logf("unable to remove repo %s: %#v", repoPath, err)
	})

	return repoPath
}

func GetNodeTypeId(env string) string {
	if env == "gcp" {
		return "n1-standard-4"
	} else if env == "aws" || env == "ucws" {
		// aws-prod-ucws has CLOUD_ENV set to "ucws"
		return "i3.xlarge"
	}
	return "Standard_DS4_v2"
}

func setupLocalFiler(t testutil.TestingT) (filer.Filer, string) {
	t.Log(testutil.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmp := t.TempDir()
	f, err := filer.NewLocalClient(tmp)
	require.NoError(t, err)

	return f, path.Join(filepath.ToSlash(tmp))
}

func setupWsfsFiler(t testutil.TestingT) (filer.Filer, string) {
	ctx, wt := acc.WorkspaceTest(t)

	tmpdir := TemporaryWorkspaceDir(t, wt.W)
	f, err := filer.NewWorkspaceFilesClient(wt.W, tmpdir)
	require.NoError(t, err)

	// Check if we can use this API here, skip test if we cannot.
	_, err = f.Read(ctx, "we_use_this_call_to_test_if_this_api_is_enabled")
	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusBadRequest {
		t.Skip(aerr.Message)
	}

	return f, tmpdir
}

func setupWsfsExtensionsFiler(t testutil.TestingT) (filer.Filer, string) {
	_, wt := acc.WorkspaceTest(t)

	tmpdir := TemporaryWorkspaceDir(t, wt.W)
	f, err := filer.NewWorkspaceFilesExtensionsClient(wt.W, tmpdir)
	require.NoError(t, err)

	return f, tmpdir
}

func setupDbfsFiler(t testutil.TestingT) (filer.Filer, string) {
	_, wt := acc.WorkspaceTest(t)

	tmpDir := TemporaryDbfsDir(t, wt.W)
	f, err := filer.NewDbfsClient(wt.W, tmpDir)
	require.NoError(t, err)

	return f, path.Join("dbfs:/", tmpDir)
}

func setupUcVolumesFiler(t testutil.TestingT) (filer.Filer, string) {
	t.Log(testutil.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	if os.Getenv("TEST_METASTORE_ID") == "" {
		t.Skip("Skipping tests that require a UC Volume when metastore id is not set.")
	}

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryUcVolume(t, w)
	f, err := filer.NewFilesClient(w, tmpDir)
	require.NoError(t, err)

	return f, path.Join("dbfs:/", tmpDir)
}
