package aircmd

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/require"
)

func TestRenderEnvelope(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	require.NoError(t, renderEnvelope(ctx, getData{RunID: "1", Status: "RUNNING"}))
}
