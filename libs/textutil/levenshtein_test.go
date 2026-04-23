package textutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "abc", 0},
		{"", "abc", 3},
		{"abc", "", 3},
		{"kitten", "sitting", 3},
		{"output", "outpu", 1},   // deletion
		{"output", "ouptut", 2},  // transposition = 2 edits
		{"output", "outpux", 1},  // substitution
		{"output", "outputx", 1}, // insertion
		{"a", "b", 1},
		{"ab", "ba", 2},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.a, tt.b), func(t *testing.T) {
			assert.Equal(t, tt.want, LevenshteinDistance(tt.a, tt.b))
			// Verify symmetry.
			assert.Equal(t, tt.want, LevenshteinDistance(tt.b, tt.a))
		})
	}
}
