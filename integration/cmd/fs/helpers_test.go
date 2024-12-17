package fs_test

import (
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/require"
)

func setupLocalFiler(t testutil.TestingT) (filer.Filer, string) {
	tmp := t.TempDir()
	f, err := filer.NewLocalClient(tmp)
	require.NoError(t, err)

	return f, path.Join(filepath.ToSlash(tmp))
}

func setupDbfsFiler(t testutil.TestingT) (filer.Filer, string) {
	_, wt := acc.WorkspaceTest(t)

	tmpdir := acc.TemporaryDbfsDir(wt)
	f, err := filer.NewDbfsClient(wt.W, tmpdir)
	require.NoError(t, err)
	return f, path.Join("dbfs:/", tmpdir)
}

func setupUcVolumesFiler(t testutil.TestingT) (filer.Filer, string) {
	_, wt := acc.WorkspaceTest(t)

	if os.Getenv("TEST_METASTORE_ID") == "" {
		t.Skip("Skipping tests that require a UC Volume when metastore id is not set.")
	}

	tmpdir := acc.TemporaryVolume(wt)
	f, err := filer.NewFilesClient(wt.W, tmpdir)
	require.NoError(t, err)

	return f, path.Join("dbfs:/", tmpdir)
}
