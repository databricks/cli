package template

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltInReader(t *testing.T) {
	exists := []string{
		"default-python",
		"default-sql",
		"dbt-sql",
	}

	for _, name := range exists {
		t.Run(name, func(t *testing.T) {
			r := &builtinReader{name: name}
			fs, err := r.FS(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, fs)

			// Assert file content returned is accurate and every template has a welcome
			// message defined.
			fd, err := fs.Open("databricks_template_schema.json")
			require.NoError(t, err)
			b, err := io.ReadAll(fd)
			require.NoError(t, err)
			assert.Contains(t, string(b), "welcome_message")
			assert.NoError(t, fd.Close())
		})
	}

	t.Run("doesnotexist", func(t *testing.T) {
		r := &builtinReader{name: "doesnotexist"}
		_, err := r.FS(context.Background())
		assert.EqualError(t, err, "builtin template doesnotexist not found")
	})
}

func TestGitUrlReader(t *testing.T) {
	ctx := context.Background()
	cmd := cmdio.NewIO(ctx, flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "", "bundles")
	ctx = cmdio.InContext(ctx, cmd)

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
	fd, err := fsys.Open("somefile")
	require.NoError(t, err)
	b, err := io.ReadAll(fd)
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))
	assert.NoError(t, fd.Close())

	// Assert the downloaded repository is cleaned up.
	fd, err = fsys.Open(".")
	require.NoError(t, err)
	require.NoError(t, fd.Close())
	r.Cleanup()
	_, err = fsys.Open(".")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestLocalReader(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(tmpDir, "somefile"), "somecontent")
	ctx := context.Background()

	r := &localReader{path: tmpDir}
	fs, err := r.FS(ctx)
	require.NoError(t, err)

	// Assert the fs returned is rooted at correct location.
	fd, err := fs.Open("somefile")
	require.NoError(t, err)
	b, err := io.ReadAll(fd)
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))
	assert.NoError(t, fd.Close())
}
