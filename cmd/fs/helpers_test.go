package fs

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilerForPathForLocalPaths(t *testing.T) {
	tmpDir := t.TempDir()
	ctx := context.Background()

	f, path, err := filerForPath(ctx, tmpDir)
	assert.NoError(t, err)
	assert.Equal(t, tmpDir, path)

	info, err := f.Stat(ctx, path)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestFilerForPathForInvalidScheme(t *testing.T) {
	ctx := context.Background()

	_, _, err := filerForPath(ctx, "dbf:/a")
	assert.ErrorContains(t, err, "invalid scheme")

	_, _, err = filerForPath(ctx, "foo:a")
	assert.ErrorContains(t, err, "invalid scheme")

	_, _, err = filerForPath(ctx, "file:/a")
	assert.ErrorContains(t, err, "invalid scheme")
}

func testWindowsFilerForPath(t *testing.T, ctx context.Context, fullPath string) {
	f, path, err := filerForPath(ctx, fullPath)
	assert.NoError(t, err)

	// Assert path remains unchanged
	assert.Equal(t, path, fullPath)

	// Assert local client is created
	_, ok := f.(*filer.LocalClient)
	assert.True(t, ok)
}

func TestFilerForWindowsLocalPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	ctx := context.Background()
	testWindowsFilerForPath(t, ctx, `c:\abc`)
	testWindowsFilerForPath(t, ctx, `c:abc`)
	testWindowsFilerForPath(t, ctx, `d:\abc`)
	testWindowsFilerForPath(t, ctx, `d:\abc`)
	testWindowsFilerForPath(t, ctx, `f:\abc\ef`)
}

func mockMustWorkspaceClientFunc(cmd *cobra.Command, args []string) error {
	return nil
}

func setupCommand(t *testing.T) (*cobra.Command, *mocks.MockWorkspaceClient) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := context.Background()
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	return cmd, m
}

func setupTest(t *testing.T) (*validArgs, *cobra.Command, *mocks.MockWorkspaceClient) {
	cmd, m := setupCommand(t)

	fakeFilerForPath := func(ctx context.Context, fullPath string) (filer.Filer, string, error) {
		fakeFiler := filer.NewFakeFiler(map[string]fakefs.FileInfo{
			"dir":       {FakeName: "root", FakeDir: true},
			"dir/dirA":  {FakeDir: true},
			"dir/dirB":  {FakeDir: true},
			"dir/fileA": {},
		})
		return fakeFiler, strings.TrimPrefix(fullPath, "dbfs:/"), nil
	}

	v := newValidArgs()
	v.filerForPathFunc = fakeFilerForPath
	v.mustWorkspaceClientFunc = mockMustWorkspaceClientFunc

	return v, cmd, m
}

func TestGetValidArgsFunctionDbfsCompletion(t *testing.T) {
	v, cmd, _ := setupTest(t)
	completions, directive := v.Validate(cmd, []string{}, "dbfs:/dir/")
	assert.Equal(t, []string{"dbfs:/dir/dirA/", "dbfs:/dir/dirB/", "dbfs:/dir/fileA"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionLocalCompletion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	v, cmd, _ := setupTest(t)
	completions, directive := v.Validate(cmd, []string{}, "dir/")
	assert.Equal(t, []string{"dir/dirA/", "dir/dirB/", "dir/fileA", "dbfs:/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionLocalCompletionWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}

	v, cmd, _ := setupTest(t)
	completions, directive := v.Validate(cmd, []string{}, "dir/")
	assert.Equal(t, []string{"dir\\dirA\\", "dir\\dirB\\", "dir\\fileA", "dbfs:/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionCompletionOnlyDirs(t *testing.T) {
	v, cmd, _ := setupTest(t)
	v.onlyDirs = true
	completions, directive := v.Validate(cmd, []string{}, "dbfs:/dir/")
	assert.Equal(t, []string{"dbfs:/dir/dirA/", "dbfs:/dir/dirB/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionNotCompletedArgument(t *testing.T) {
	cmd, _ := setupCommand(t)

	v := newValidArgs()
	v.pathArgCount = 0
	v.mustWorkspaceClientFunc = mockMustWorkspaceClientFunc

	completions, directive := v.Validate(cmd, []string{}, "dbfs:/")

	assert.Nil(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}
