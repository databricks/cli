package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestPatternTrie_SearchPath(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		mustMatch    []string
		mustNotMatch []string
	}{
		{
			name:         "empty pattern",
			pattern:      "",
			mustMatch:    []string{""},
			mustNotMatch: []string{"foo"},
		},
		{
			name:         "simple key pattern",
			pattern:      "foo",
			mustMatch:    []string{"foo"},
			mustNotMatch: []string{"foo.bar", "foo[0]", "bar"},
		},

		{
			name:         "nested key pattern",
			pattern:      "foo.bar",
			mustMatch:    []string{"foo.bar"},
			mustNotMatch: []string{"foo", "foo[0]", "bar.foo", "foo.baz"},
		},
		{
			name:         "root wildcard",
			pattern:      "*",
			mustMatch:    []string{"foo", "bar"},
			mustNotMatch: []string{"", "bar.foo", "foo.baz"},
		},
		{
			name:         "wildcard * after foo",
			pattern:      "foo.*",
			mustMatch:    []string{"foo.bar", "foo.baz"},
			mustNotMatch: []string{"foo", "bar", "foo.bar.baz"},
		},
		{
			name:         "wildcard [*] after foo",
			pattern:      "foo[*]",
			mustMatch:    []string{"foo[0]", "foo[1]", "foo[2025]"},
			mustNotMatch: []string{"foo", "bar", "foo[0].bar"},
		},
		{
			name:         "key after * wildcard",
			pattern:      "foo.*.bar",
			mustMatch:    []string{"foo.abc.bar", "foo.def.bar"},
			mustNotMatch: []string{"foo", "bar", "foo.bar.baz"},
		},
		{
			name:         "key after [*] wildcard",
			pattern:      "foo[*].bar",
			mustMatch:    []string{"foo[0].bar", "foo[1].bar", "foo[2025].bar"},
			mustNotMatch: []string{"foo", "bar", "foo[0].baz"},
		},
		{
			name:         "multiple * wildcards",
			pattern:      "*.*.*",
			mustMatch:    []string{"foo.bar.baz", "foo.bar.qux"},
			mustNotMatch: []string{"foo", "bar", "foo.bar", "foo.bar.baz.qux"},
		},
		{
			name:         "multiple [*] wildcards",
			pattern:      "foo[*][*]",
			mustMatch:    []string{"foo[0][0]", "foo[1][1]", "foo[2025][2025]"},
			mustNotMatch: []string{"foo", "bar", "foo[0][0][0]"},
		},
		{
			name:         "[*] after * wildcard",
			pattern:      "*[*]",
			mustMatch:    []string{"foo[0]", "foo[1]", "foo[2025]"},
			mustNotMatch: []string{"foo", "bar", "foo[0].bar", "[0].foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := &dyn.TrieNode{}
			pattern := dyn.MustPatternFromString(tt.pattern)

			// None of the expected paths should match yet.
			for _, path := range tt.mustMatch {
				_, ok := trie.SearchPath(dyn.MustPathFromString(path))
				assert.False(t, ok)
			}
			for _, path := range tt.mustNotMatch {
				_, ok := trie.SearchPath(dyn.MustPathFromString(path))
				assert.False(t, ok)
			}

			err := trie.Insert(pattern)
			assert.NoError(t, err)

			// Now all the expected paths should match.
			for _, path := range tt.mustMatch {
				pattern, ok := trie.SearchPath(dyn.MustPathFromString(path))
				assert.True(t, ok)
				assert.Equal(t, dyn.MustPatternFromString(tt.pattern), pattern)
			}
			for _, path := range tt.mustNotMatch {
				_, ok := trie.SearchPath(dyn.MustPathFromString(path))
				assert.False(t, ok)
			}
		})
	}
}

func TestPatternTrie_MultiplePatterns(t *testing.T) {
	trie := &dyn.TrieNode{}

	patterns := []string{
		"foo.bar",
		"foo.*.baz",
		"def[*]",
	}

	mustMatch := map[string]string{
		"foo.bar":     "foo.bar",
		"foo.abc.baz": "foo.*.baz",
		"foo.def.baz": "foo.*.baz",
		"def[0]":      "def[*]",
		"def[1]":      "def[*]",
	}

	mustNotMatch := []string{
		"foo",
		"abc[0]",
		"abc[1]",
		"def[2].x",
		"foo.y",
		"foo.bar.baz.qux",
	}

	for _, pattern := range patterns {
		err := trie.Insert(dyn.MustPatternFromString(pattern))
		assert.NoError(t, err)
	}

	for path, expectedPattern := range mustMatch {
		pattern, ok := trie.SearchPath(dyn.MustPathFromString(path))
		assert.True(t, ok)
		assert.Equal(t, dyn.MustPatternFromString(expectedPattern), pattern)
	}

	for _, path := range mustNotMatch {
		_, ok := trie.SearchPath(dyn.MustPathFromString(path))
		assert.False(t, ok)
	}
}

func TestPatternTrie_OverlappingPatterns(t *testing.T) {
	trie := &dyn.TrieNode{}

	// Insert overlapping patterns
	patterns := []string{
		"foo.bar",
		"foo.*",
		"*.bar",
		"*.*",
	}

	for _, pattern := range patterns {
		err := trie.Insert(dyn.MustPatternFromString(pattern))
		assert.NoError(t, err)
	}

	for _, path := range []string{
		"foo.bar",
		"foo.baz",
		"baz.bar",
		"baz.qux",
	} {
		_, ok := trie.SearchPath(dyn.MustPathFromString(path))
		assert.True(t, ok)
	}
}

func TestPatternTrie_FixedIndexPatterns(t *testing.T) {
	trie := &dyn.TrieNode{}

	err := trie.Insert(dyn.MustPatternFromString("foo[0]"))
	assert.EqualError(t, err, "fixed index patterns are not supported: dyn.Pattern{dyn.pathComponent{key:\"foo\", index:0}, dyn.pathComponent{key:\"\", index:0}}")

	err = trie.Insert(dyn.MustPatternFromString("foo[2]"))
	assert.EqualError(t, err, "fixed index patterns are not supported: dyn.Pattern{dyn.pathComponent{key:\"foo\", index:0}, dyn.pathComponent{key:\"\", index:2}}")
}
