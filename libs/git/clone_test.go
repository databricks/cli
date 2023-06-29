package git

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
)

func TestGitCloneCLINotFound(t *testing.T) {
	// Set PATH to "", so git CLI cannot be found
	t.Setenv("PATH", "")
	
	tmpDir := t.TempDir()
	cmdIO := cmdio.NewIO("text", os.Stdin, os.Stdout, os.Stderr, "")
	ctx := cmdio.InContext(context.Background(), cmdIO)

	err := Clone(ctx, CloneOptions{
		Provider:       "github",
		Organization:   "databricks",
		RepositoryName: "does-not-exist",
		Reference:      "main",
		TargetDir:      tmpDir,
	})
	assert.ErrorIs(t, err, exec.ErrNotFound)
	assert.ErrorContains(t, err, "please install git CLI to download private templates")
}
