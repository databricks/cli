package bundletest

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

// This took 40ms to run on 18th June 2025.
func BenchmarkWalkReadOnly(b *testing.B) {
	input := BundleV(b, 10000)

	for b.Loop() {
		err := dyn.WalkReadOnly(input, func(p dyn.Path, v dyn.Value) error {
			return nil
		})
		assert.NoError(b, err)
	}
}

// This took 160ms to run on 18th June 2025.
func BenchmarkWalk(b *testing.B) {
	input := BundleV(b, 10000)

	for b.Loop() {
		_, err := dyn.Walk(input, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return v, nil
		})
		assert.NoError(b, err)
	}
}

// This took 49 microseconds to run on 6th Aug 2025.
func BenchmarkEnumPrefixTree(b *testing.B) {
	for b.Loop() {
		// Generate prefix tree for all enum fields.
		trie := &dyn.TrieNode{}
		for k := range generated.EnumFields {
			pattern, err := dyn.NewPatternFromString(k)
			assert.NoError(b, err)

			err = trie.Insert(pattern)
			assert.NoError(b, err)
		}
	}
}

// This took 15 microseconds to run on 6th Aug 2025.
func BenchmarkRequiredPrefixTree(b *testing.B) {
	for b.Loop() {
		// Generate prefix tree for all required fields.
		trie := &dyn.TrieNode{}
		for k := range generated.RequiredFields {
			pattern, err := dyn.NewPatternFromString(k)
			assert.NoError(b, err)

			err = trie.Insert(pattern)
			assert.NoError(b, err)
		}
	}
}
