package completer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCompleter(t *testing.T, onlyDirs bool) *completer {
	ctx := t.Context()
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

	// The typed "dir/" is kept so the completions still extend it.
	assert.Equal(t, []string{"dir/dirA\\", "dir/dirB\\", "dbfs:/"}, completions)
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

// TestFilerCompleterLocalPartialName uses the OS separator, so on Windows it covers
// "dir\\di": the listed directory must come from the same split as the typed prefix.
func TestFilerCompleterLocalPartialName(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dir", "dirA"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dir", "dirB"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "dir", "fileA"), nil, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dirC"), 0o755))
	t.Chdir(root)

	localFiler, err := filer.NewLocalClient("")
	require.NoError(t, err)
	c := New(t.Context(), localFiler, true)
	c.SetIsLocalPath(true)

	sep := string(filepath.Separator)
	completions, directive, err := c.CompletePath("dir" + sep + "di")
	require.NoError(t, err)
	assert.Equal(t, []string{"dir" + sep + "dirA" + sep, "dir" + sep + "dirB" + sep, "dbfs:/"}, completions)
	assert.Equal(t, cobra.ShellCompDirectiveNoSpace, directive)
}
