package internal

import (
	"bytes"
	"context"
	"io"
	"path"
	"regexp"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceImportDir(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())
	tmpdir := temporaryWorkspaceDir(t, w)

	// run import-dir command
	RequireSuccessfulRun(t, "workspace", "import-dir", "./testdata/import_dir/default", tmpdir)

	// assert files are uploaded
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)
	assertFileContains(t, ctx, f, "foo.txt", "hello, world")
	assertFileContains(t, ctx, f, ".gitignore", ".databricks")
	assertFileContains(t, ctx, f, "bar/apple.py", "print(1)")
	assertNotebookExists(t, ctx, w, path.Join(tmpdir, "bar/mango"))
}

func TestWorkspaceImportDirOverwriteFlag(t *testing.T) {
	// t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	// ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())
	tmpdir := temporaryWorkspaceDir(t, w)

	// run import-dir command
	RequireSuccessfulRun(t, "workspace", "import-dir", "./testdata/import_dir/override/a", tmpdir)

	// // assert files are uploaded
	// f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	// require.NoError(t, err)
	// assertFileContains(t, ctx, f, "bar.txt", "from directory A")

	// Assert another run fails with path already exists error from the server
	_, _, err := RequireErrorRun(t, "workspace", "import-dir", "./testdata/import_dir/override/b", tmpdir)
	assert.Regexp(t, regexp.MustCompile("Path (.*) already exists."), err.Error())

	// // Succeeds with the overwrite flag
	// RequireSuccessfulRun(t, "workspace", "import-dir", "./testdata/import_dir/override/b", tmpdir, "--overwrite")
	// require.NoError(t, err)
	// assertFileContains(t, ctx, f, "bar.txt", "from directory B")
}

func assertFileContains(t *testing.T, ctx context.Context, f filer.Filer, name, contents string) {
	r, err := f.Read(ctx, name)
	require.NoError(t, err)

	var b bytes.Buffer
	_, err = io.Copy(&b, r)
	require.NoError(t, err)

	assert.Contains(t, b.String(), contents)
}

func assertNotebookExists(t *testing.T, ctx context.Context, w *databricks.WorkspaceClient, path string) {
	info, err := w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
		Path: path,
	})
	require.NoError(t, err)
	assert.Len(t, info, 1)
	assert.Equal(t, info[0].ObjectType, workspace.ObjectTypeNotebook)
}
