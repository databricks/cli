package agentstream

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		contains []string
		absent   []string
	}{
		{
			name:     "keeps content of query embedded blocks",
			in:       "Some text\n<!-- begin-embedded:query_abc -->\n| A | B |\n| --- | --- |\n| 1 | 2 |\n<!-- end-embedded:query_abc -->\nMore text",
			contains: []string{"Some text", "| A | B |", "More text"},
			absent:   []string{"begin-embedded", "end-embedded"},
		},
		{
			name:     "removes viz embedded blocks entirely",
			in:       "Before\n<!-- begin-embedded:viz_123 -->\n![Chart](#viz_123)\n<!-- end-embedded:viz_123 -->\nAfter",
			contains: []string{"Before", "After"},
			absent:   []string{"viz_123", "Chart", "begin-embedded"},
		},
		{
			name:     "removes standalone viz image references",
			in:       "Intro\n![Total by Franchise](#viz_abc)\nOutro",
			contains: []string{"Intro", "Outro"},
			absent:   []string{"viz_abc", "Total by Franchise"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderMarkdown(&buf, tc.in)
			for _, s := range tc.contains {
				assert.Contains(t, buf.String(), s)
			}
			for _, s := range tc.absent {
				assert.NotContains(t, buf.String(), s)
			}
		})
	}
}
