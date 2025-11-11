package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipReadmeHeadingAndParagraph(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "no heading",
			input:    "Some content without heading",
			expected: "Some content without heading",
		},
		{
			name:     "heading only",
			input:    "# Heading",
			expected: "# Heading",
		},
		{
			name: "heading with paragraph",
			input: `# Heading

This is the first paragraph.

This is the second paragraph.`,
			expected: "This is the second paragraph.",
		},
		{
			name: "heading with empty lines before paragraph",
			input: `# Heading


First paragraph.

Second paragraph.`,
			expected: "Second paragraph.",
		},
		{
			name: "heading with multiple paragraphs",
			input: `# Heading

First paragraph with multiple lines.
Still first paragraph.

Second paragraph.

Third paragraph.`,
			expected: `Still first paragraph.

Second paragraph.

Third paragraph.`,
		},
		{
			name: "heading at end",
			input: `# Heading
First paragraph.`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := skipReadmeHeadingAndParagraph(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
