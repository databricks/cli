package filer_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
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
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok && aerr.StatusCode == http.StatusBadRequest {
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
	ctx, wt := acc.WorkspaceTest(t)

	if os.Getenv("TEST_METASTORE_ID") == "" {
		t.Skip("Skipping tests that require a UC Volume when metastore id is not set.")
	}

	tmpdir := acc.TemporaryVolume(wt)
	f, err := filer.NewFilesClient(ctx, wt.W, tmpdir)
	require.NoError(t, err)

	return f, path.Join("dbfs:/", tmpdir)
}

func setupSandboxFiler(t testutil.TestingT) (filer.Filer, string) {
	ctx, wt := acc.WorkspaceTest(t)

	sandboxID := os.Getenv("TEST_SANDBOX_ID")
	if sandboxID == "" {
		t.Skip("Skipping tests that require a sandbox when TEST_SANDBOX_ID is not set.")
	}

	parent := "sandboxes/" + sandboxID
	tmpdir := "/tmp/" + testutil.RandomName("filer-it-")
	_, err := wt.W.Sandbox.CreateDirectory(ctx, sandbox.CreateDirectoryRequest{
		Parent:  parent,
		Path:    tmpdir,
		Parents: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := wt.W.Sandbox.DeletePath(context.WithoutCancel(ctx), sandbox.DeletePathRequest{
			Parent:    parent,
			Path:      tmpdir,
			Recursive: true,
		})
		if err != nil {
			t.Logf("Unable to remove temporary sandbox directory %s: %#v", tmpdir, err)
		}
	})

	f, err := filer.NewSandboxClient(wt.W, sandboxID, tmpdir)
	require.NoError(t, err)
	return f, tmpdir
}
