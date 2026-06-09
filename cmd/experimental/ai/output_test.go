package ai

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/require"
)

func TestRenderEnvelope(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	require.NoError(t, renderEnvelope(ctx, statusData{RunID: "1", Status: "RUNNING"}))
}
