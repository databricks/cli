package databrickscfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHost(t *testing.T) {
	assert.Equal(t, "invalid", normalizeHost("invalid"))

	// With port.
	assert.Equal(t, "http://foo:123", normalizeHost("http://foo:123"))
	assert.Equal(t, "https://foo:80", normalizeHost("https://foo:80"))

	// With the scheme's default port.
	assert.Equal(t, "https://foo", normalizeHost("https://foo:443"))
	assert.Equal(t, "http://foo", normalizeHost("http://foo:80/"))
	assert.Equal(t, "https://[::1]", normalizeHost("https://[::1]:443"))

	// With trailing slash.
	assert.Equal(t, "http://foo", normalizeHost("http://foo/"))

	// With path.
	assert.Equal(t, "http://foo", normalizeHost("http://foo/bar"))

	// With query string.
	assert.Equal(t, "http://foo", normalizeHost("http://foo?bar"))

	// With anchor.
	assert.Equal(t, "http://foo", normalizeHost("http://foo#bar"))
}
