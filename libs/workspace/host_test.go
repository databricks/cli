package workspace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHost(t *testing.T) {
	assert.Equal(t, "invalid", NormalizeHost("invalid"))

	// With port.
	assert.Equal(t, "http://foo:123", NormalizeHost("http://foo:123"))

	// With trailing slash.
	assert.Equal(t, "http://foo", NormalizeHost("http://foo/"))

	// With path.
	assert.Equal(t, "http://foo", NormalizeHost("http://foo/bar"))

	// With query string.
	assert.Equal(t, "http://foo", NormalizeHost("http://foo?bar"))

	// With anchor.
	assert.Equal(t, "http://foo", NormalizeHost("http://foo#bar"))
}
