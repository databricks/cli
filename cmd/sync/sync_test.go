package sync

import (
	"context"
	"testing"

	"github.com/databricks/bricks/cmd/root"
	"github.com/stretchr/testify/assert"
)

func TestItSyncs(t *testing.T) {
	ctx := context.Background()
	root.RootCmd.SetArgs([]string{"sync"})
	err := root.RootCmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}
