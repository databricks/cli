package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "value with quotes",
			input:    `say "hello"`,
			expected: `say \"hello\"`,
		},
		{
			name:     "value with newline",
			input:    "line1\nline2",
			expected: "line1line2",
		},
		{
			name:     "value with carriage return",
			input:    "line1\rline2",
			expected: "line1line2",
		},
		{
			name:     "value with CRLF",
			input:    "line1\r\nline2",
			expected: "line1line2",
		},
		{
			name:     "value with quotes and newlines",
			input:    "say \"hello\"\nworld",
			expected: `say \"hello\"world`,
		},
		{
			name:     "empty value",
			input:    "",
			expected: "",
		},
		{
			name:     "only newlines",
			input:    "\n\r\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeEnvValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
