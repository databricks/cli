package databrickscfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHost(t *testing.T) {
	assert.Equal(t, "invalid", normalizeHost("invalid"))

	// With port.
	assert.Equal(t, "http://foo:123", normalizeHost("http://foo:123"))

	// With trailing slash.
	assert.Equal(t, "http://foo", normalizeHost("http://foo/"))

	// With path.
	assert.Equal(t, "http://foo", normalizeHost("http://foo/bar"))

	// With query string.
	assert.Equal(t, "http://foo", normalizeHost("http://foo?bar"))

	// With anchor.
	assert.Equal(t, "http://foo", normalizeHost("http://foo#bar"))
}

func TestSameHost(t *testing.T) {
	assert.True(t, SameHost("https://foo.example.com", "https://foo.example.com"))

	// Trailing slash and path are ignored.
	assert.True(t, SameHost("https://foo.example.com", "https://foo.example.com/"))
	assert.True(t, SameHost("https://foo.example.com", "https://foo.example.com/bar"))

	// Different hosts.
	assert.False(t, SameHost("https://foo.example.com", "https://bar.example.com"))

	// Different scheme.
	assert.False(t, SameHost("https://foo.example.com", "http://foo.example.com"))
}
