package template

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultWriterConfigure(t *testing.T) {
	// Test on local file system.
	w := &defaultWriter{}
	err := w.Configure(context.Background(), "/foo/bar", "/out/abc")
	assert.NoError(t, err)

	assert.Equal(t, "/foo/bar", w.configPath)
	assert.IsType(t, &filer.LocalClient{}, w.outputFiler)
}

func TestDefaultWriterConfigureOnDBR(t *testing.T) {
	// This test is not valid on windows because a DBR image is always based on
	// Linux.
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	ctx := dbr.MockRuntime(context.Background(), dbr.Environment{IsDbr: true, Version: "15.4"})
	ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{Host: "https://myhost.com"},
	})
	w := &defaultWriter{}
	err := w.Configure(ctx, "/foo/bar", "/Workspace/out/abc")
	assert.NoError(t, err)

	assert.Equal(t, "/foo/bar", w.configPath)
	assert.IsType(t, &filer.WorkspaceFilesExtensionsClient{}, w.outputFiler)
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

func TestGenerateFileTree(t *testing.T) {
	testFiles := []file{
		&inMemoryFile{relPath: "README.md"},
		&inMemoryFile{relPath: "src/main.go"},
		&inMemoryFile{relPath: "src/utils.go"},
		&inMemoryFile{relPath: "config/app.yaml"},
		&inMemoryFile{relPath: "config/dev.yaml"},
	}

	tree := generateFileTree(testFiles)

	// Check that root files are included
	assert.Contains(t, tree, "README.md")

	// Check that directories are shown with trailing slash
	assert.Contains(t, tree, "src/")
	assert.Contains(t, tree, "config/")

	// Check that files in directories are indented
	assert.Contains(t, tree, "  main.go")
	assert.Contains(t, tree, "  utils.go")
	assert.Contains(t, tree, "  app.yaml")
	assert.Contains(t, tree, "  dev.yaml")
}

func TestGenerateFileTreeWithManyFiles(t *testing.T) {
	// Create more than 10 files in a directory
	testFiles := []file{
		&inMemoryFile{relPath: "README.md"},
	}
	for i := range 15 {
		testFiles = append(testFiles, &inMemoryFile{relPath: "src/file" + string(rune(i)) + ".go"})
	}

	tree := generateFileTree(testFiles)

	// Check that directory is shown but files are collapsed
	assert.Contains(t, tree, "src/")
	assert.Contains(t, tree, "(15 files)")

	// Individual files should not be listed
	assert.NotContains(t, tree, "  file0.go")
}
