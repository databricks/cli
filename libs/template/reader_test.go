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
			fsys, err := r.FS(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, fsys)

			// Assert file content returned is accurate and every template has a welcome
			// message defined.
			b, err := fs.ReadFile(fsys, "databricks_template_schema.json")
			require.NoError(t, err)
			assert.Contains(t, string(b), "welcome_message")
		})
	}

	t.Run("doesnotexist", func(t *testing.T) {
		r := &builtinReader{name: "doesnotexist"}
		_, err := r.FS(context.Background())
		assert.EqualError(t, err, "builtin template doesnotexist not found")
	})
}

func TestGitUrlReader(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())

	var args []string
	numCalls := 0
	cloneFunc := func(ctx context.Context, url, reference, targetPath string) error {
		numCalls++
		args = []string{url, reference, targetPath}
		testutil.WriteFile(t, filepath.Join(targetPath, "a", "b", "c", "somefile"), "somecontent")
		return nil
	}
	r := &gitReader{
		gitUrl:      "someurl",
		cloneFunc:   cloneFunc,
		ref:         "sometag",
		templateDir: "a/b/c",
	}

	// Assert cloneFunc is called with the correct args.
	fsys, err := r.FS(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, r.tmpRepoDir)
	assert.Equal(t, 1, numCalls)
	assert.DirExists(t, r.tmpRepoDir)
	assert.Equal(t, []string{"someurl", "sometag", r.tmpRepoDir}, args)

	// Assert the fs returned is rooted at the templateDir.
	b, err := fs.ReadFile(fsys, "somefile")
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))

	// Assert second call to FS returns an error.
	_, err = r.FS(ctx)
	assert.ErrorContains(t, err, "FS called twice on git reader")

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
	ctx := context.Background()

	r := &localReader{path: tmpDir}
	fsys, err := r.FS(ctx)
	require.NoError(t, err)

	// Assert the fs returned is rooted at correct location.
	b, err := fs.ReadFile(fsys, "somefile")
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))
}
