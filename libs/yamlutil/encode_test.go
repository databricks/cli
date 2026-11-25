package yamlutil

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeKeepsLongLinesAndIndentsByTwo(t *testing.T) {
	// Longer than 80 columns: a line width limit would fold it onto several lines.
	description := strings.Repeat("word ", 30) + "end"

	var buf bytes.Buffer
	err := Encode(&buf, map[string]any{
		"job": map[string]any{
			"description": description,
			"tasks":       []string{"a"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "job:\n  description: "+description+"\n  tasks:\n    - a\n", buf.String())
}
