package labs_test

import (
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowMissingProjectReturnsNotFound(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	r := testcli.NewRunner(t, ctx, "labs", "show", "missing")
	_, _, err := r.Run()
	require.Error(t, err)
	assert.ErrorIs(t, err, apierr.ErrNotFound)
}
