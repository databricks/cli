package jsonsaver

import (
	"testing"

	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestEncoder_MarshalNoEscape(t *testing.T) {
	out, err := marshalNoEscape("1 < 2")
	if !assert.NoError(t, err) {
		return
	}

	// Confirm the output.
	assert.JSONEq(t, `"1 < 2"`, string(out))

	// Confirm that HTML escaping is disabled.
	assert.NotContains(t, string(out), "\\u003c")

	// Confirm that the encoder writes a trailing newline.
	assert.Contains(t, string(out), "\n")
}

func TestEncoder_MarshalIndentNoEscape(t *testing.T) {
	out, err := marshalIndentNoEscape([]string{"1 < 2", "2 < 3"}, "", "  ")
	if !assert.NoError(t, err) {
		return
	}

	// Confirm the output.
	assert.JSONEq(t, `["1 < 2", "2 < 3"]`, string(out))

	// Confirm that HTML escaping is disabled.
	assert.NotContains(t, string(out), "\\u003c")

	// Confirm that the encoder performs indenting and writes a trailing newline.
	assert.Contains(t, string(out), "[\n")
	assert.Contains(t, string(out), "  \"1 < 2\",\n")
	assert.Contains(t, string(out), "]\n")
}
