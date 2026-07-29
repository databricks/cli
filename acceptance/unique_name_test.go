package acceptance_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIUniqueName(t *testing.T) {
	// 26 lowercase base32 characters, like the generated unique name.
	random := "osr5mzrrvzb73juixjoviti24y"

	// Prefix prepended, same length as input, lowercase-alphanumeric.
	assert.Equal(t, "ci15799017600xabcosr5mzrrv", ciUniqueName("ci15799017600xabc", random))
	assert.Equal(t, "ci1xabcosr5mzrrvzb73juixjo", ciUniqueName("ci1xabc", random))

	// Empty prefix: unchanged.
	assert.Equal(t, random, ciUniqueName("", random))

	// Prefix that leaves exactly the 8-char random minimum: prepended.
	assert.Equal(t, "ci123456789012xabcosr5mzrr", ciUniqueName("ci123456789012xabc", random))

	// Prefix too long to leave enough randomness: unchanged.
	assert.Equal(t, random, ciUniqueName("ci1234567890123xabc", random))
}
