package filer_test

import (
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/require"
)

func setupLocalFiler(t testutil.TestingT) (filer.Filer, string) {
	tmp := t.TempDir()
	f, err := filer.NewLocalClient(tmp)
	require.NoError(t, err)

	return f, path.Join(filepath.ToSlash(tmp))
}

func setupWsfsFiler(t testutil.TestingT) (filer.Filer, string) {
	ctx, wt := acc.WorkspaceTest(t)

	tmpdir := acc.TemporaryWorkspaceDir(wt)
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

	tmpdir := acc.TemporaryWorkspaceDir(wt)
	f, err := filer.NewWorkspaceFilesExtensionsClient(wt.W, tmpdir)
	require.NoError(t, err)
	return f, tmpdir
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
