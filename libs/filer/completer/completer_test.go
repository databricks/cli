package completer

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func setupCompleter(t *testing.T, onlyDirs bool) *completer {
	ctx := context.Background()
	// Needed to make type context.valueCtx for mockFilerForPath
	ctx = cmdctx.SetWorkspaceClient(ctx, mocks.NewMockWorkspaceClient(t).WorkspaceClient)

	fakeFiler := filer.NewFakeFiler(map[string]fakefs.FileInfo{
		"dir":       {FakeName: "root", FakeDir: true},
		"dir/dirA":  {FakeDir: true},
		"dir/dirB":  {FakeDir: true},
		"dir/fileA": {},
	})

	completer := New(ctx, fakeFiler, onlyDirs)
	completer.SetIsLocalPath(false)
	return completer
}

func TestFilerCompleterSetsPrefix(t *testing.T) {
	completer := setupCompleter(t, true)
	completer.SetPrefix("dbfs:")
	completions, directive, err := completer.CompletePath("dir/")

	assert.Equal(t, []string{"dbfs:/dir/dirA/", "dbfs:/dir/dirB/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.NoError(t, err)
}

func TestFilerCompleterReturnsNestedDirs(t *testing.T) {
	completer := setupCompleter(t, true)
	completions, directive, err := completer.CompletePath("dir/")

	assert.Equal(t, []string{"dir/dirA/", "dir/dirB/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.NoError(t, err)
}

func TestFilerCompleterReturnsAdjacentDirs(t *testing.T) {
	completer := setupCompleter(t, true)
	completions, directive, err := completer.CompletePath("dir/wrong_path")

	assert.Equal(t, []string{"dir/dirA/", "dir/dirB/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.NoError(t, err)
}

func TestFilerCompleterReturnsNestedDirsAndFiles(t *testing.T) {
	completer := setupCompleter(t, false)
	completions, directive, err := completer.CompletePath("dir/")

	assert.Equal(t, []string{"dir/dirA/", "dir/dirB/", "dir/fileA"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.NoError(t, err)
}

func TestFilerCompleterAddsDbfsPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	completer := setupCompleter(t, true)
	completer.SetIsLocalPath(true)
	completions, directive, err := completer.CompletePath("dir/")

	assert.Equal(t, []string{"dir/dirA/", "dir/dirB/", "dbfs:/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.NoError(t, err)
}

func TestFilerCompleterWindowsSeparator(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}

	completer := setupCompleter(t, true)
	completer.SetIsLocalPath(true)
	completions, directive, err := completer.CompletePath("dir/")

	assert.Equal(t, []string{"dir\\dirA\\", "dir\\dirB\\", "dbfs:/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
	assert.NoError(t, err)
}

func TestFilerCompleterNoCompletions(t *testing.T) {
	completer := setupCompleter(t, true)
	completions, directive, err := completer.CompletePath("wrong_dir/wrong_dir")

	assert.Nil(t, completions)
	assert.Equal(t, cobra.ShellCompDirectiveError, directive)
	assert.Error(t, err)
}
