package bundlecleanup_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/databricks/cli/acceptance/internal/bundlecleanup"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type fakeLister struct {
	children map[string][]workspace.ObjectInfo
	errs     map[string]error
}

func (f fakeLister) ListAll(_ context.Context, request workspace.ListWorkspaceRequest) ([]workspace.ObjectInfo, error) {
	if err := f.errs[request.Path]; err != nil {
		return nil, err
	}
	return f.children[request.Path], nil
}

func TestValidatePrefix(t *testing.T) {
	assert.NoError(t, bundlecleanup.ValidatePrefix("ci123x", "123"))
	assert.Error(t, bundlecleanup.ValidatePrefix("ci123x", "456"))
	assert.Error(t, bundlecleanup.ValidatePrefix("ci123x", "invalid"))
	assert.Error(t, bundlecleanup.ValidatePrefix("ci123x", ""))
	assert.NoError(t, bundlecleanup.ValidatePrefix("localmabc123abcd", ""))
	assert.Error(t, bundlecleanup.ValidatePrefix("local", ""))
	assert.Error(t, bundlecleanup.ValidatePrefix("ci123xabcd", ""))
	assert.Error(t, bundlecleanup.ValidatePrefix("localmabc123ABCD", ""))
}

func TestNewBundleNamePrefix(t *testing.T) {
	t.Setenv("GITHUB_RUN_ID", "123")
	prefix, err := bundlecleanup.NewBundleNamePrefix(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "ci123x", prefix)

	t.Setenv("GITHUB_RUN_ID", "")
	prefix, err = bundlecleanup.NewBundleNamePrefix(t.Context())
	require.NoError(t, err)
	assert.Regexp(t, `^local[0-9a-z]+[0-9a-z]{4}$`, prefix)
	assert.NoError(t, bundlecleanup.ValidatePrefix(prefix, ""))
}

func TestFindDeploymentRoots(t *testing.T) {
	lister := fakeLister{children: map[string][]workspace.ObjectInfo{
		"/root": {
			{Path: "/root/default", ObjectType: workspace.ObjectTypeDirectory},
			{Path: "/root/nested", ObjectType: workspace.ObjectTypeDirectory},
			{Path: "/root/file.txt", ObjectType: workspace.ObjectTypeFile},
		},
		"/root/default": {
			{Path: "/root/default/state", ObjectType: workspace.ObjectTypeDirectory},
			{Path: "/root/default/files", ObjectType: workspace.ObjectTypeDirectory},
		},
		"/root/nested": {
			{Path: "/root/nested/app", ObjectType: workspace.ObjectTypeDirectory},
		},
		"/root/nested/app": {
			{Path: "/root/nested/app/files", ObjectType: workspace.ObjectTypeDirectory},
		},
	}}
	assert.Equal(t, []string{"/root/default", "/root/nested/app"}, bundlecleanup.FindDeploymentRoots(t.Context(), lister, "/root"))
}

func TestListChildDirsIgnoresNotFound(t *testing.T) {
	lister := fakeLister{errs: map[string]error{"/missing": apierr.ErrNotFound}}
	assert.Empty(t, bundlecleanup.ListChildDirs(t.Context(), lister, "/missing"))

	lister.errs["/broken"] = errors.New("broken")
	assert.Empty(t, bundlecleanup.ListChildDirs(t.Context(), lister, "/broken"))
}

func TestDestroyBundle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a shell script")
	}
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args")
	require.NoError(t, os.WriteFile(argsPath, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0o755))

	out, err := bundlecleanup.DestroyBundle(argsPath, "/Workspace/test-bundle")
	require.NoError(t, err)
	assert.Equal(t, "bundle\ndestroy\n--target\ndefault\n--auto-approve\n--force-lock\n", string(out))
}
