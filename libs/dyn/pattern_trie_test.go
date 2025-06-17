package dyn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternTrie_SearchPath(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		expected    []string
		notExpected []string
	}{
		{
			name:        "empty pattern",
			pattern:     "",
			expected:    []string{""},
			notExpected: []string{"foo"},
		},
		{
			name:        "simple key pattern",
			pattern:     "foo",
			expected:    []string{"foo"},
			notExpected: []string{"foo.bar", "foo[0]", "bar"},
		},
		{
			name:        "simple index pattern",
			pattern:     "[0]",
			expected:    []string{"[0]"},
			notExpected: []string{"foo[0]", "foo.bar", "bar"},
		},
		{
			name:        "nested key pattern",
			pattern:     "foo.bar",
			expected:    []string{"foo.bar"},
			notExpected: []string{"foo", "foo[0]", "bar.foo", "foo.baz"},
		},
		{
			name:        "root wildcard",
			pattern:     "*",
			expected:    []string{"foo", "bar"},
			notExpected: []string{"", "bar.foo", "foo.baz"},
		},
		{
			name:        "wildcard * after foo",
			pattern:     "foo.*",
			expected:    []string{"foo.bar", "foo.baz"},
			notExpected: []string{"foo", "bar", "foo.bar.baz"},
		},
		{
			name:        "wildcard [*] after foo",
			pattern:     "foo[*]",
			expected:    []string{"foo[0]", "foo[1]", "foo[2025]"},
			notExpected: []string{"foo", "bar", "foo[0].bar"},
		},
		{
			name:        "key after * wildcard",
			pattern:     "foo.*.bar",
			expected:    []string{"foo.abc.bar", "foo.def.bar"},
			notExpected: []string{"foo", "bar", "foo.bar.baz"},
		},
		{
			name:        "key after [*] wildcard",
			pattern:     "foo[*].bar",
			expected:    []string{"foo[0].bar", "foo[1].bar", "foo[2025].bar"},
			notExpected: []string{"foo", "bar", "foo[0].baz"},
		},
		{
			name:        "multiple * wildcards",
			pattern:     "*.*.*",
			expected:    []string{"foo.bar.baz", "foo.bar.qux"},
			notExpected: []string{"foo", "bar", "foo.bar", "foo.bar.baz.qux"},
		},
		{
			name:        "multiple [*] wildcards",
			pattern:     "foo[*][*]",
			expected:    []string{"foo[0][0]", "foo[1][1]", "foo[2025][2025]"},
			notExpected: []string{"foo", "bar", "foo[0][0][0]"},
		},
		{
			name:        "[*] after * wildcard",
			pattern:     "*[*]",
			expected:    []string{"foo[0]", "foo[1]", "foo[2025]"},
			notExpected: []string{"foo", "bar", "foo[0].bar", "[0].foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewPatternTrie()
			pattern := MustPatternFromString(tt.pattern)

			// None of the expected paths should match yet.
			for _, path := range tt.expected {
				_, ok := trie.SearchPath(MustPathFromString(path))
				assert.False(t, ok)
			}
			for _, path := range tt.notExpected {
				_, ok := trie.SearchPath(MustPathFromString(path))
				assert.False(t, ok)
			}

			err := trie.Insert(pattern)
			assert.NoError(t, err)

			// Now all the expected paths should match.
			for _, path := range tt.expected {
				pattern, ok := trie.SearchPath(MustPathFromString(path))
				assert.True(t, ok)
				assert.Equal(t, MustPatternFromString(tt.pattern), pattern)
			}
			for _, path := range tt.notExpected {
				_, ok := trie.SearchPath(MustPathFromString(path))
				assert.False(t, ok)
			}
		})
	}
}

func TestPatternTrie_MultiplePatterns(t *testing.T) {
	trie := NewPatternTrie()

	patterns := []string{
		"foo.bar",
		"foo.*.baz",
		"abc[0]",
		"def[*]",
	}

	expected := map[string]string{
		"foo.bar":     "foo.bar",
		"foo.abc.baz": "foo.*.baz",
		"foo.def.baz": "foo.*.baz",
		"abc[0]":      "abc[0]",
		"def[0]":      "def[*]",
		"def[1]":      "def[*]",
	}

	notExpected := []string{
		"foo",
		"abc[1]",
		"def[2].x",
		"foo.y",
		"foo.bar.baz.qux",
	}

	for _, pattern := range patterns {
		err := trie.Insert(MustPatternFromString(pattern))
		assert.NoError(t, err)
	}

	for path, expectedPattern := range expected {
		pattern, ok := trie.SearchPath(MustPathFromString(path))
		assert.True(t, ok)
		assert.Equal(t, MustPatternFromString(expectedPattern), pattern)
	}

	for _, path := range notExpected {
		_, ok := trie.SearchPath(MustPathFromString(path))
		assert.False(t, ok)
	}
}

func TestPatternTrie_OverlappingPatterns(t *testing.T) {
	trie := NewPatternTrie()

	// Insert overlapping patterns
	patterns := []string{
		"foo.bar",
		"foo.*",
		"*.bar",
		"*.*",
	}

	for _, pattern := range patterns {
		err := trie.Insert(MustPatternFromString(pattern))
		assert.NoError(t, err)
	}

	for _, path := range []string{
		"foo.bar",
		"foo.baz",
		"baz.bar",
		"baz.qux",
	} {
		_, ok := trie.SearchPath(MustPathFromString(path))
		assert.True(t, ok)
	}
}
