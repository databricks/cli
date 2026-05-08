package cmdiotest_test

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelect_EmptyItems pins the early-error path: cmdio.Select rejects an
// empty item list before drawing any prompt UI. No pty needed; the function
// returns immediately with a "expected to have <label>" error.
func TestSelect_EmptyItems(t *testing.T) {
	ctx := t.Context()
	var buf bytes.Buffer
	io := cmdio.NewIO(ctx, flags.OutputText, &bytes.Buffer{}, &buf, &buf, "", "")
	ctx = cmdio.InContext(ctx, io)

	id, err := cmdio.SelectOrdered(ctx, nil, "a workspace")
	require.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "expected to have a workspace")
	assert.Empty(t, buf.String(), "no UI should render for an empty list")
}
