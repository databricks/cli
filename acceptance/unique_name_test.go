package acceptance_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIUniqueName(t *testing.T) {
	// 26 lowercase base32 characters, like the generated unique name.
	random := "osr5mzrrvzb73juixjoviti24y"

	// Run id and leg suffix embedded, same length as input, lowercase-alphanumeric, sweepable prefix.
	assert.Equal(t, "ci15799017600xabcosr5mzrrv", ciUniqueName("15799017600", "abc", random))
	assert.Equal(t, "ci1xabcosr5mzrrvzb73juixjo", ciUniqueName("1", "abc", random))

	// Empty leg suffix (backward-compatible prefix) still works.
	assert.Equal(t, "ci15799017600xosr5mzrrvzb7", ciUniqueName("15799017600", "", random))

	// No or invalid run id: unchanged.
	assert.Equal(t, random, ciUniqueName("", "abc", random))
	assert.Equal(t, random, ciUniqueName("abc123", "abc", random))
	assert.Equal(t, random, ciUniqueName("123 456", "abc", random))

	// 12-digit run id plus the 3-char suffix leaves exactly the 8-char random minimum: prefixed.
	assert.Equal(t, "ci123456789012xabcosr5mzrr", ciUniqueName("123456789012", "abc", random))

	// 13-digit run id plus the suffix is too long to leave enough randomness: unchanged.
	assert.Equal(t, random, ciUniqueName("1234567890123", "abc", random))
}
