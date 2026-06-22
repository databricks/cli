package acceptance_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIUniqueName(t *testing.T) {
	// 26 lowercase base32 characters, like the generated unique name.
	random := "osr5mzrrvzb73juixjoviti24y"

	// Run id embedded, same length as input, lowercase-alphanumeric, sweepable prefix.
	assert.Equal(t, "ci15799017600xosr5mzrrvzb7", ciUniqueName("15799017600", random))
	assert.Equal(t, "ci1xosr5mzrrvzb73juixjovit", ciUniqueName("1", random))

	// No or invalid run id: unchanged.
	assert.Equal(t, random, ciUniqueName("", random))
	assert.Equal(t, random, ciUniqueName("abc123", random))
	assert.Equal(t, random, ciUniqueName("123 456", random))

	// 15-digit run id still leaves exactly the 8-char random minimum: prefixed.
	assert.Equal(t, "ci123456789012345xosr5mzrr", ciUniqueName("123456789012345", random))

	// 16-digit run id is too long to leave enough randomness: unchanged.
	assert.Equal(t, random, ciUniqueName("1234567890123456", random))
}
