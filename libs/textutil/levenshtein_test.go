package textutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		dist int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "ab", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"my_cluster_id", "my_clster_id", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.dist, Levenshtein(tt.a, tt.b))
			// Symmetric.
			assert.Equal(t, tt.dist, Levenshtein(tt.b, tt.a))
		})
	}
}

func TestClosestMatch(t *testing.T) {
	candidates := []string{"cluster_id", "cluster_name", "node_type"}

	t.Run("close_match", func(t *testing.T) {
		match, dist := ClosestMatch("cluser_id", candidates)
		assert.Equal(t, "cluster_id", match)
		assert.Equal(t, 1, dist)
	})

	t.Run("no_match", func(t *testing.T) {
		match, _ := ClosestMatch("zzzzzzz", candidates)
		assert.Equal(t, "", match)
	})

	t.Run("exact_match", func(t *testing.T) {
		match, dist := ClosestMatch("cluster_id", candidates)
		assert.Equal(t, "cluster_id", match)
		assert.Equal(t, 0, dist)
	})

	t.Run("short_key_threshold", func(t *testing.T) {
		// For a 2-char key, threshold = min(3, max(1, 1)) = 1
		match, _ := ClosestMatch("ab", []string{"ac", "zz"})
		assert.Equal(t, "ac", match)
	})
}
