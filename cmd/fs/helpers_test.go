package fs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func mockCurrentUserApi(m *mocks.MockWorkspaceClient, u *iam.User, e error) {
	currentUserApi := m.GetMockCurrentUserAPI()
	currentUserApi.EXPECT().Me(mock.AnythingOfType("*context.valueCtx")).Return(u, e)
}

func mockMustWorkspaceClientFunc(cmd *cobra.Command, args []string) error {
	return nil
}

func setupCommand(t *testing.T) (*cobra.Command, *mocks.MockWorkspaceClient) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := context.Background()
	ctx = root.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)

	return cmd, m
}

func setupTest(t *testing.T, path string) (*validArgs, *cobra.Command, *mocks.MockWorkspaceClient) {
	cmd, m := setupCommand(t)

	mockCurrentUserApi(m, nil, nil)

	mockFilerForPath := testutil.GetMockFilerForPath(t, path, []fs.DirEntry{
		testutil.NewFakeDirEntry("dir", true),
		testutil.NewFakeDirEntry("file", false),
	})

	v := newValidArgs()
	v.filerForPathFunc = mockFilerForPath
	v.mustWorkspaceClientFunc = mockMustWorkspaceClientFunc

	return v, cmd, m
}

func TestGetValidArgsFunctionCompletion(t *testing.T) {
	v, cmd, _ := setupTest(t, "/")

	completions, directive := v.Validate(cmd, []string{}, "dbfs:/")

	// dbfs:/Volumes is programmatically added to the completions
	assert.Equal(t, []string{"dbfs:/dir", "dbfs:/file", "dbfs:/Volumes"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionCompletionOnlyDirs(t *testing.T) {
	v, cmd, _ := setupTest(t, "/")

	v.onlyDirs = true
	completions, directive := v.Validate(cmd, []string{}, "dbfs:/")

	assert.Equal(t, []string{"dbfs:/dir", "dbfs:/Volumes"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionNoPath(t *testing.T) {
	v, cmd, _ := setupTest(t, "")

	completions, directive := v.Validate(cmd, []string{}, "")

	// Suggest both dbfs and local paths at beginning of completion
	assert.Equal(t, []string{"dir", "file", "dbfs:/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionLocalPath(t *testing.T) {
	v, cmd, _ := setupTest(t, "")

	completions, directive := v.Validate(cmd, []string{}, "d")

	assert.Equal(t, []string{"dir", "file", "dbfs:/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionAbsoluteLocalPath(t *testing.T) {
	v, cmd, _ := setupTest(t, "")

	completions, directive := v.Validate(cmd, []string{}, "/")

	assert.Equal(t, []string{"dir", "file"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionAddsSeparator(t *testing.T) {
	v, cmd, _ := setupTest(t, "parent_dir")

	completions, directive := v.Validate(cmd, []string{}, "parent_dir")

	separator := "/"
	if runtime.GOOS == "windows" {
		separator = "\\"
	}

	assert.Equal(t, []string{
		fmt.Sprintf("%s%s%s", "parent_dir", separator, "dir"),
		fmt.Sprintf("%s%s%s", "parent_dir", separator, "file"),
	}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestGetValidArgsFunctionNoCurrentUser(t *testing.T) {
	cmd, m := setupCommand(t)

	mockCurrentUserApi(m, nil, errors.New("Current User Error"))

	v := newValidArgs()
	v.mustWorkspaceClientFunc = mockMustWorkspaceClientFunc

	completions, directive := v.Validate(cmd, []string{}, "dbfs:/")

	assert.Nil(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveError, directive)
}

func TestGetValidArgsFunctionNotCompletedArgument(t *testing.T) {
	cmd, m := setupCommand(t)

	mockCurrentUserApi(m, nil, nil)

	v := newValidArgs()
	v.pathArgCount = 0
	v.mustWorkspaceClientFunc = mockMustWorkspaceClientFunc

	completions, directive := v.Validate(cmd, []string{}, "dbfs:/")

	assert.Nil(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}
