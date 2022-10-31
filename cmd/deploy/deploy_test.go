package build

import (
	"context"
	"os"
	"testing"

	"github.com/databricks/bricks/cmd/root"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	t.Skip()
	ctx := context.Background()
	os.Setenv("BRICKS_ROOT", "/Users/serge.smertin/git/labs/transpiler")
	root.RootCmd.SetArgs([]string{"deploy"})

	err := root.RootCmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}

func TestArcuate(t *testing.T) {
	t.Skip()
	ctx := context.Background()
	os.Setenv("BRICKS_ROOT", "/Users/serge.smertin/git/labs/arcuate")
	root.RootCmd.SetArgs([]string{"deploy"})

	err := root.RootCmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}
