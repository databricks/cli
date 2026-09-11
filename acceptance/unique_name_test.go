package acceptance_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCIUniqueName(t *testing.T) {
	// 26 lowercase base32 characters, like the generated unique name.
	random := "osr5mzrrvzb73juixjoviti24y"

	// Prefix prepended, same length as input, lowercase-alphanumeric.
	assert.Equal(t, "ci15799017600xabcdosr5mzrr", ciUniqueName("ci15799017600xabcd", random))
	assert.Equal(t, "ci1xabcdosr5mzrrvzb73juixj", ciUniqueName("ci1xabcd", random))

	// Empty prefix (off cloud): unchanged.
	assert.Equal(t, random, ciUniqueName("", random))

	// An 11-digit run id plus the 4-char suffix leaves exactly the 8-char random minimum: prepended.
	assert.Equal(t, "ci15799017600xabcdosr5mzrr", ciUniqueName("ci15799017600xabcd", random))

	// A prefix too long to leave 8 random chars is a bug in newBundleNamePrefix, so it panics.
	assert.Panics(t, func() { ciUniqueName("ci159990176001xabcd", random) })
}
