package template

import (
	"context"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWriterConfigure(t *testing.T) {
	// Test on local file system.
	w1 := &defaultWriter{}
	err := w1.Configure(context.Background(), "/foo/bar", "/out/abc")
	assert.NoError(t, err)

	assert.Equal(t, "/foo/bar", w1.configPath)
	assert.IsType(t, &filer.LocalClient{}, w1.outputFiler)

	// Test on DBR
	ctx := dbr.MockRuntime(context.Background(), true)
	ctx = root.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.com"},
	})
	w2 := &defaultWriter{}
	err = w2.Configure(ctx, "/foo/bar", "/Workspace/out/abc")
	assert.NoError(t, err)

	assert.Equal(t, "/foo/bar", w2.configPath)
	assert.IsType(t, &filer.WorkspaceFilesClient{}, w2.outputFiler)
}

func TestMaterializeForNonTemplateDirectory(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()
	ctx := context.Background()

	w := &defaultWriter{}
	err := w.Configure(ctx, "/foo/bar", tmpDir1)
	require.NoError(t, err)

	// Try to materialize a non-template directory.
	err = w.Materialize(ctx, &localReader{path: tmpDir2})
	assert.EqualError(t, err, "not a bundle template: expected to find a template schema file at databricks_template_schema.json")
}
