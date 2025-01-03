package template

import (
	"context"
	"io"
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
		r := &builtinReader{name: name}
		fs, err := r.FS(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, fs)
	}

	// TODO: Read one of the files to confirm further test this reader.
	r := &builtinReader{name: "doesnotexist"}
	_, err := r.FS(context.Background())
	assert.EqualError(t, err, "builtin template doesnotexist not found")
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
		err := os.MkdirAll(filepath.Join(targetPath, "a/b/c"), 0o755)
		require.NoError(t, err)
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
	fs, err := r.FS(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, r.tmpRepoDir)
	assert.DirExists(t, r.tmpRepoDir)
	assert.Equal(t, []string{"someurl", "sometag", r.tmpRepoDir}, args)

	// Assert the fs returned is rooted at the templateDir.
	fd, err := fs.Open("somefile")
	require.NoError(t, err)
	defer fd.Close()
	b, err := io.ReadAll(fd)
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))

	// Assert the FS is cached. cloneFunc should not be called again.
	_, err = r.FS(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, numCalls)

	// Assert Close cleans up the tmpRepoDir.
	err = r.Close()
	require.NoError(t, err)
	assert.NoDirExists(t, r.tmpRepoDir)
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
	defer fd.Close()
	b, err := io.ReadAll(fd)
	require.NoError(t, err)
	assert.Equal(t, "somecontent", string(b))

	// Assert close does not error
	assert.NoError(t, r.Close())
}

func TestFailReader(t *testing.T) {
	r := &failReader{}
	assert.Error(t, r.Close())
	_, err := r.FS(context.Background())
	assert.Error(t, err)
}
