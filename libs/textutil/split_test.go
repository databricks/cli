package textutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output []string
	}{
		{
			name:   "empty input",
			input:  "",
			output: nil,
		},
		{
			name:   "single scope",
			input:  "all-apis",
			output: []string{"all-apis"},
		},
		{
			name:   "trims whitespace",
			input:  " all-apis , sql ",
			output: []string{"all-apis", "sql"},
		},
		{
			name:   "drops empty entries",
			input:  "all-apis, ,sql,,",
			output: []string{"all-apis", "sql"},
		},
		{
			name:   "only empty entries",
			input:  " , , ",
			output: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.output, SplitAndTrim(tt.input))
		})
	}
}
