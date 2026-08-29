package cmdiotest_test

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelect_NoPromptSupport pins the early-error path: when the cmdIO can't
// prompt (non-TTY streams, no DATABRICKS_OUTPUT_FORMAT override),
// cmdio.SelectOrdered returns "expected to have <label>" without drawing any
// UI. This is the path CI and other non-interactive callers hit.
func TestSelect_NoPromptSupport(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	var buf bytes.Buffer
	io := cmdio.NewIO(ctx, flags.OutputText, &bytes.Buffer{}, &buf, &buf, "", "")
	ctx = cmdio.InContext(ctx, io)

	id, err := cmdio.SelectOrdered(ctx, []cmdio.Tuple{{Name: "alpha", Id: "a"}}, "a workspace")
	require.Error(t, err)
	assert.Empty(t, id)
	assert.EqualError(t, err, "expected to have a workspace")
	assert.Empty(t, buf.String(), "no UI should render without prompt support")
}
