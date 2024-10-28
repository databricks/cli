package jsonsaver

import (
	"strings"
	"testing"

	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestEncoder_marshalNoEscape(t *testing.T) {
	out, err := marshalNoEscape("1 < 2")
	require.NoError(t, err)

	// Confirm the output.
	assert.JSONEq(t, `"1 < 2"`, string(out))

	// Confirm that HTML escaping is disabled.
	assert.False(t, strings.Contains(string(out), "\\u003c"))

	// Confirm that the encoder writes a trailing newline.
	assert.True(t, strings.HasSuffix(string(out), "\n"))
}

func TestEncoder_marshalIndentNoEscape(t *testing.T) {
	out, err := marshalIndentNoEscape([]string{"1 < 2", "2 < 3"}, "", "  ")
	require.NoError(t, err)

	// Confirm the output.
	assert.JSONEq(t, `["1 < 2", "2 < 3"]`, string(out))

	// Confirm that HTML escaping is disabled.
	assert.False(t, strings.Contains(string(out), "\\u003c"))

	// Confirm that the encoder performs indenting and writes a trailing newline.
	assert.True(t, strings.HasPrefix(string(out), "[\n"))
	assert.True(t, strings.Contains(string(out), "  \"1 < 2\",\n"))
	assert.True(t, strings.HasSuffix(string(out), "]\n"))
}
