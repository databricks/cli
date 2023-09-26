package bundle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

func initTestTemplate(t *testing.T, templateName string, config map[string]any) (string, error) {
	templateRoot := filepath.Join("bundles", templateName)

	bundleRoot := t.TempDir()
	configFilePath, err := writeConfigFile(t, config)
	if err != nil {
		return "", err
	}

	ctx := root.SetWorkspaceClient(context.Background(), nil)
	cmd := cmdio.NewIO(flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "bundles")
	ctx = cmdio.InContext(ctx, cmd)

	err = template.Materialize(ctx, configFilePath, templateRoot, bundleRoot)
	return bundleRoot, err
}

func writeConfigFile(t *testing.T, config map[string]any) (string, error) {
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	dir := t.TempDir()
	filepath := filepath.Join(dir, "config.json")
	t.Log("Configuration for template: ", string(bytes))

	err = os.WriteFile(filepath, bytes, 0644)
	return filepath, err
}

func deployBundle(t *testing.T, path string) error {
	t.Setenv("BUNDLE_ROOT", path)
	c := internal.NewCobraTestRunner(t, "bundle", "deploy", "--force-lock")
	_, _, err := c.Run()
	return err
}

func runResource(t *testing.T, path string, key string) (string, error) {
	ctx := context.Background()
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "run", key)
	stdout, _, err := c.Run()
	return stdout.String(), err
}

func destroyBundle(t *testing.T, path string) error {
	t.Setenv("BUNDLE_ROOT", path)
	c := internal.NewCobraTestRunner(t, "bundle", "destroy", "--auto-approve")
	_, _, err := c.Run()
	return err
}

var sparkVersions = []string{
	"11.3.x-scala2.12",
	"12.2.x-scala2.12",
	"13.0.x-scala2.12",
	"13.1.x-scala2.12",
	"13.2.x-scala2.12",
	"13.3.x-scala2.12",
	"14.0.x-scala2.12",
	"14.1.x-scala2.12",
}

func generateNotebookTasks(notebookPath string, nodeTypeId string) []jobs.SubmitTask {
	tasks := make([]jobs.SubmitTask, 0)
	for i := 0; i < len(sparkVersions); i++ {
		task := jobs.SubmitTask{
			TaskKey: fmt.Sprintf("notebook_%s", strings.ReplaceAll(sparkVersions[i], ".", "_")),
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: notebookPath,
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     sparkVersions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func generateSparkPythonTasks(notebookPath string, nodeTypeId string) []jobs.SubmitTask {
	tasks := make([]jobs.SubmitTask, 0)
	for i := 0; i < len(sparkVersions); i++ {
		task := jobs.SubmitTask{
			TaskKey: fmt.Sprintf("spark_%s", strings.ReplaceAll(sparkVersions[i], ".", "_")),
			SparkPythonTask: &jobs.SparkPythonTask{
				PythonFile: notebookPath,
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     sparkVersions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func generateWheelTasks(wheelPath string, nodeTypeId string, versions []string) []jobs.SubmitTask {
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

func temporaryWorkspaceDir(t *testing.T, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	basePath := fmt.Sprintf("/Users/%s/%s", me.UserName, internal.RandomName("integration-test-python-"))

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

func temporaryDbfsDir(t *testing.T, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	path := fmt.Sprintf("/tmp/%s", internal.RandomName("integration-test-dbfs-"))

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

func temporaryRepo(t *testing.T, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, internal.RandomName("integration-test-repo-"))

	t.Logf("Creating repo:%s", repoPath)
	repoInfo, err := w.Repos.Create(ctx, workspace.CreateRepo{
		Url:      "https://github.com/andrewnester/python-info",
		Provider: "github",
		Path:     repoPath,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Logf("Removing DVFS folder:%s", repoPath)
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

func getNodeTypeId(env string) string {
	if env == "gcp" {
		return "n1-standard-4"
	} else if env == "aws" {
		return "i3.xlarge"
	}
	return "Standard_DS4_v2"
}
