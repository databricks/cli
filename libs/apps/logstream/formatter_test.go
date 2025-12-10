package logstream

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatter_FormatEntry(t *testing.T) {
	entry := &wsEntry{Source: "app", Timestamp: 1705315800.0, Message: "hello world\n"}

	t.Run("json output", func(t *testing.T) {
		jsonFormatter := newLogFormatter(false, flags.OutputJSON)
		output := jsonFormatter.FormatEntry(entry)

		var parsed wsEntry
		require.NoError(t, json.Unmarshal([]byte(output), &parsed))

		assert.Equal(t, "APP", parsed.Source)
		assert.Greater(t, parsed.Timestamp, 0.0)
		assert.Equal(t, "hello world", parsed.Message)
		assert.NotContains(t, output, "\x1b[")
	})

	t.Run("text output", func(t *testing.T) {
		textFormatter := newLogFormatter(false, flags.OutputText)
		output := textFormatter.FormatEntry(entry)

		assert.Contains(t, output, "[APP]")
		assert.Contains(t, output, "hello world")
		assert.Contains(t, output, "2024-01-15")
		assert.NotContains(t, output, "\x1b[")
	})
}
