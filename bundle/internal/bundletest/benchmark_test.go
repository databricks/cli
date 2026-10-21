package bundletest

import (
	"testing"

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
