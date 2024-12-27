package labs_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/require"
)

func TestListingWorks(t *testing.T) {
	ctx := context.Background()
	ctx = env.WithUserHomeDir(ctx, "project/testdata/installed-in-home")
	c := testcli.NewRunner(t, ctx, "labs", "list")
	stdout, _, err := c.Run()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ucx")
}
