package completer

import (
	"context"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func setupCompleter(t *testing.T, onlyDirs bool) *completer {
	ctx := context.Background()
	// Needed to make type context.valueCtx for mockFilerForPath
	ctx = root.SetWorkspaceClient(ctx, mocks.NewMockWorkspaceClient(t).WorkspaceClient)

	fakeFiler := filer.NewFakeFiler(map[string]filer.FakeFileInfo{
		"dir":       {FakeName: "root", FakeDir: true},
		"dir/dirA":  {FakeDir: true},
		"dir/dirB":  {FakeDir: true},
		"dir/fileA": {},
	})

	return New(ctx, fakeFiler, onlyDirs)
}

func TestFilerCompleterReturnsNestedDirs(t *testing.T) {
	completer := setupCompleter(t, true)
	completions, dirPath, directive, err := completer.CompletePath("dir")

	assert.Equal(t, []string{"dirA", "dirB"}, completions)
	assert.Equal(t, "dir", dirPath)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.Nil(t, err)
}

func TestFilerCompleterReturnsAdjacentDirs(t *testing.T) {
	completer := setupCompleter(t, true)
	completions, dirPath, directive, err := completer.CompletePath("dir/wrong_dir")

	assert.Equal(t, []string{"dirA", "dirB"}, completions)
	assert.Equal(t, "dir", dirPath)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.Nil(t, err)
}

func TestFilerCompleterReturnsNestedDirsAndFiles(t *testing.T) {
	completer := setupCompleter(t, false)
	completions, dirPath, directive, err := completer.CompletePath("dir")

	assert.Equal(t, []string{"dirA", "dirB", "fileA"}, completions)
	assert.Equal(t, "dir", dirPath)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.Nil(t, err)
}

func TestFilerCompleterNoCompletions(t *testing.T) {
	completer := setupCompleter(t, true)
	completions, dirPath, directive, err := completer.CompletePath("wrong_dir/wrong_dir")

	assert.Nil(t, completions)
	assert.Equal(t, "wrong_dir", dirPath)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.Nil(t, err)
}

// func TestFilerCompleterReadDirError(t *testing.T) {
