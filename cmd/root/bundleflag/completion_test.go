package bundleflag

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func prepareCompletionTest(t *testing.T, dir string) *cobra.Command {
	testutil.CleanupEnvironment(t)
	testutil.Chdir(t, dir)
	cmd := cobra.Command{}
	cmd.SetContext(context.Background())
	return &cmd
}

func TestCompletionEmpty(t *testing.T) {
	cmd := prepareCompletionTest(t, "./testdata/empty")
	out, comp := targetCompletion(cmd, []string{}, "")
	assert.Equal(t, cobra.ShellCompDirectiveError, comp)
	assert.Empty(t, out)
}

func TestCompletionInvalid(t *testing.T) {
	cmd := prepareCompletionTest(t, "./testdata/invalid")
	out, comp := targetCompletion(cmd, []string{}, "")
	assert.Equal(t, cobra.ShellCompDirectiveError, comp)
	assert.Empty(t, out)
}

func TestCompletionValid(t *testing.T) {
	cmd := prepareCompletionTest(t, "./testdata/valid")
	out, comp := targetCompletion(cmd, []string{}, "")
	assert.Equal(t, cobra.ShellCompDirectiveDefault, comp)
	assert.ElementsMatch(t, []string{"foo", "bar", "qux"}, out)
}
