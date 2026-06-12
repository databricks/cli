package acceptance_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIUniqueName(t *testing.T) {
	// 26 lowercase base32 characters, like the generated unique name.
	random := "osr5mzrrvzb73juixjoviti24y"

	// Run id embedded, same length as input, sweepable prefix.
	assert.Equal(t, "ci-15799017600-osr5mzrrvzb", ciUniqueName("15799017600", random))
	assert.Equal(t, "ci-1-osr5mzrrvzb73juixjovi", ciUniqueName("1", random))

	// No or invalid run id: unchanged.
	assert.Equal(t, random, ciUniqueName("", random))
	assert.Equal(t, random, ciUniqueName("abc123", random))
	assert.Equal(t, random, ciUniqueName("123 456", random))

	// Run id too long to leave enough randomness: unchanged.
	assert.Equal(t, random, ciUniqueName("123456789012345", random))
}
