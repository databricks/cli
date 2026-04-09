package aitools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple", "a,b", []string{"a", "b"}},
		{"whitespace", "a, b , c", []string{"a", "b", "c"}},
		{"empty input", "", nil},
		{"trailing comma", "a,b,", []string{"a", "b"}},
		{"only commas", ",,", nil},
		{"single value", "a", []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, splitAndTrim(tt.input))
		})
	}
}
