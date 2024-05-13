package python

import (
	"testing"

	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

type parseLogEntryTestCase struct {
	text     string
	ok       bool
	expected logEntry
}

func TestParseLogEntry(t *testing.T) {

	testCases := []parseLogEntryTestCase{
		{
			text: "{\"level\":\"INFO\",\"message\":\"hello\"}",
			ok:   true,
			expected: logEntry{
				Level:   "INFO",
				Message: "hello",
			},
		},
		{
			text: "{\"level\":\"DEBUG\",\"message\":\"hello\"}",
			ok:   true,
			expected: logEntry{
				Level:   "DEBUG",
				Message: "hello",
			},
		},
		{
			text: "{\"level\":\"WARNING\",\"message\":\"hello\"}",
			ok:   true,
			expected: logEntry{
				Level:   "WARNING",
				Message: "hello",
			},
		},
		{
			text: "{\"level\":\"ERROR\",\"message\":\"hello\"}",
			ok:   true,
			expected: logEntry{
				Level:   "ERROR",
				Message: "hello",
			},
		},
		{
			text:     "hi {} there",
			ok:       false,
			expected: logEntry{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.text, func(t *testing.T) {
			actual, ok := parseLogEntry(tc.text)

			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
