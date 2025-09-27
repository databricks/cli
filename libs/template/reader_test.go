package template

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltInReader(t *testing.T) {
	exists := []string{
		"default-python",
		"default-sql",
		"dbt-sql",
		"experimental-jobs-as-code",
	}

	for _, name := range exists {
		t.Run(name, func(t *testing.T) {
			r := &builtinReader{name: name}
			schema, fsys, err := r.LoadSchemaAndTemplateFS(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, fsys)
			assert.NotNil(t, schema)

			// Assert schema has a welcome message defined.
			assert.NotEmpty(t, schema.WelcomeMessage)
		})
	}

	t.Run("doesnotexist", func(t *testing.T) {
		r := &builtinReader{name: "doesnotexist"}
		_, _, err := r.LoadSchemaAndTemplateFS(context.Background())
		assert.EqualError(t, err, "builtin template doesnotexist not found")
	})
}

func TestBuiltInReaderTemplateDir(t *testing.T) {
	// Test that template_dir property works correctly
	// default-python template should use schema from default-python/ but template files from default/
	r := &builtinReader{name: "default-python"}
	schema, fsys, err := r.LoadSchemaAndTemplateFS(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, schema)
	assert.NotNil(t, fsys)

	// Verify the schema contains default-python specific content
	assert.Contains(t, schema.WelcomeMessage, "default Python template")

	// Verify we can read template files (should come from default/)
	templateFiles, err := fs.ReadDir(fsys, "template")
	require.NoError(t, err)
	assert.NotEmpty(t, templateFiles)

	// Verify that a specific template file exists (this should come from default/ template)
	_, err = fs.Stat(fsys, "template/{{.project_name}}/databricks.yml.tmpl")
	assert.NoError(t, err)

	// Test that a template without template_dir works normally
	r2 := &builtinReader{name: "default-sql"}
	schema2, fsys2, err := r2.LoadSchemaAndTemplateFS(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, schema2)
	assert.NotNil(t, fsys2)

	// For default-sql, the schema should not reference template_dir
	assert.Contains(t, schema2.WelcomeMessage, "default SQL template")

	// Verify that lakeflow-pipelines also uses template_dir correctly
	r3 := &builtinReader{name: "lakeflow-pipelines"}
	schema3, fsys3, err := r3.LoadSchemaAndTemplateFS(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, schema3)
	assert.NotNil(t, fsys3)

	// lakeflow-pipelines should also have template files from default/
	_, err = fs.Stat(fsys3, "template/{{.project_name}}/databricks.yml.tmpl")
	assert.NoError(t, err)
}

func TestGitUrlReader(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())

	var args []string
	numCalls := 0
	cloneFunc := func(ctx context.Context, url, reference, targetPath string) error {
		numCalls++
		args = []string{url, reference, targetPath}
		testutil.WriteFile(t, filepath.Join(targetPath, "a", "b", "c", "somefile"), "somecontent")
		testutil.WriteFile(t, filepath.Join(targetPath, "a", "b", "c", "databricks_template_schema.json"), `{"welcome_message": "test"}`)
		return nil
	}
	r := &gitReader{
		gitUrl:      "someurl",
		cloneFunc:   cloneFunc,
		ref:         "sometag",
		templateDir: "a/b/c",
	}

	// Assert cloneFunc is called with the correct args.
	schema, fsys, err := r.LoadSchemaAndTemplateFS(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, r.tmpRepoDir)
	assert.Equal(t, 1, numCalls)
	assert.DirExists(t, r.tmpRepoDir)
	assert.Equal(t, []string{"someurl", "sometag", r.tmpRepoDir}, args)
	assert.NotNil(t, schema)

	// Assert the fs returned is rooted at the templateDir.
	b, err := fs.ReadFile(fsys, "somefile")
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))

	// Assert second call returns an error.
	_, _, err = r.LoadSchemaAndTemplateFS(ctx)
	assert.ErrorContains(t, err, "LoadSchemaAndTemplateFS called twice on git reader")

	// Assert the downloaded repository is cleaned up.
	_, err = fs.Stat(fsys, ".")
	require.NoError(t, err)
	r.Cleanup(ctx)
	_, err = fs.Stat(fsys, ".")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestLocalReader(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(tmpDir, "somefile"), "somecontent")
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks_template_schema.json"), `{"welcome_message": "test"}`)
	ctx := context.Background()

	r := &localReader{path: tmpDir}
	schema, fsys, err := r.LoadSchemaAndTemplateFS(ctx)
	require.NoError(t, err)
	assert.NotNil(t, schema)

	// Assert the fs returned is rooted at correct location.
	b, err := fs.ReadFile(fsys, "somefile")
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))
}
