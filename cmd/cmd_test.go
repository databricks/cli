package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCommandNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, spec := range topLevelSpecs() {
		assert.False(t, seen[spec.Name()], "duplicate top-level command %q", spec.Name())
		seen[spec.Name()] = true
	}
}
