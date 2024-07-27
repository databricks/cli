package completer

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/databricks/cli/cmd/root"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setup(t *testing.T) context.Context {
	ctx := context.Background()
	// Needed to make type context.valueCtx for mockFilerForPath
	ctx = root.SetWorkspaceClient(ctx, mocks.NewMockWorkspaceClient(t).WorkspaceClient)
	return ctx
}

func TestFilerCompleterReturnsNestedDirs(t *testing.T) {
	ctx := setup(t)

	mockFilerForPath := testutil.GetMockFilerForPath(t, "/", []fs.DirEntry{
		testutil.NewFakeDirEntry("dir", true),
	})
	mockFiler, _, _ := mockFilerForPath(ctx, "dbfs:/")

	completer := NewCompleter(ctx, mockFiler, true)

	completions, directive := completer.CompleteRemotePath("/")

	assert.Equal(t, []string{"/dir"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestFilerCompleterReturnsAdjacentDirs(t *testing.T) {
	ctx := setup(t)

	mockFiler := mockfiler.NewMockFiler(t)

	// First call to get nested dirs fails so we get the adjacent dirs instead
	mockFiler.On("ReadDir", mock.AnythingOfType("*context.valueCtx"), "/wrong_path").Return(nil, errors.New("error"))
	mockFiler.On("ReadDir", mock.AnythingOfType("*context.valueCtx"), "/").Return([]fs.DirEntry{
		testutil.NewFakeDirEntry("adjacent", true),
	}, nil)

	completer := NewCompleter(ctx, mockFiler, true)
	completions, directive := completer.CompleteRemotePath("/wrong_path")

	assert.Equal(t, []string{"/adjacent"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)

	mockFiler.AssertExpectations(t)
}

func TestFilerCompleterReadDirError(t *testing.T) {
	ctx := setup(t)

	mockFiler := mockfiler.NewMockFiler(t)
	mockFiler.On("ReadDir", mock.AnythingOfType("*context.valueCtx"), mock.Anything).Return(nil, errors.New("error"))

	completer := NewCompleter(ctx, mockFiler, true)
	completions, directive := completer.CompleteRemotePath("/dir")

	assert.Nil(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)

	mockFiler.AssertExpectations(t)
}

func TestFilerCompleterReturnsFileAndDir(t *testing.T) {
	ctx := setup(t)

	mockFilerForPath := testutil.GetMockFilerForPath(t, "/", []fs.DirEntry{
		testutil.NewFakeDirEntry("dir", true),
		testutil.NewFakeDirEntry("file", false),
	})
	mockFiler, _, _ := mockFilerForPath(ctx, "dbfs:/")

	// onlyDirs=false so we should get both files and directories
	completer := NewCompleter(ctx, mockFiler, false)

	completions, directive := completer.CompleteRemotePath("/")

	assert.Equal(t, []string{"/dir", "/file"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestFilerCompleterRetainsFormatting(t *testing.T) {
	ctx := setup(t)

	mockFilerForPath := testutil.GetMockFilerForPath(t, "//dir//", []fs.DirEntry{
		testutil.NewFakeDirEntry("nested_dir", true),
	})
	mockFiler, _, _ := mockFilerForPath(ctx, "dbfs://dir")

	completer := NewCompleter(ctx, mockFiler, true)

	completions, directive := completer.CompleteRemotePath("//dir//")

	assert.Equal(t, []string{"//dir//nested_dir"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}

func TestFilerCompleterAddsSeparator(t *testing.T) {
	ctx := setup(t)

	mockFilerForPath := testutil.GetMockFilerForPath(t, "/dir", []fs.DirEntry{
		testutil.NewFakeDirEntry("nested_dir", true),
	})
	mockFiler, _, _ := mockFilerForPath(ctx, "dbfs:/dir")

	completer := NewCompleter(ctx, mockFiler, true)

	completions, directive := completer.CompleteRemotePath("/dir")

	assert.Equal(t, []string{"/dir/nested_dir"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}
