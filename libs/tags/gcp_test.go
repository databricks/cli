package tags

import (
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func TestGcpOuter(t *testing.T) {
	assert.True(t, unicode.In('A', gcpOuter))
	assert.True(t, unicode.In('Z', gcpOuter))
	assert.True(t, unicode.In('a', gcpOuter))
	assert.True(t, unicode.In('z', gcpOuter))
	assert.True(t, unicode.In('0', gcpOuter))
	assert.True(t, unicode.In('9', gcpOuter))
	assert.False(t, unicode.In('-', gcpOuter))
	assert.False(t, unicode.In('.', gcpOuter))
	assert.False(t, unicode.In('_', gcpOuter))
	assert.False(t, unicode.In('!', gcpOuter))
}

func TestGcpInner(t *testing.T) {
	assert.True(t, unicode.In('A', gcpInner))
	assert.True(t, unicode.In('Z', gcpInner))
	assert.True(t, unicode.In('a', gcpInner))
	assert.True(t, unicode.In('z', gcpInner))
	assert.True(t, unicode.In('0', gcpInner))
	assert.True(t, unicode.In('9', gcpInner))
	assert.True(t, unicode.In('-', gcpInner))
	assert.True(t, unicode.In('.', gcpInner))
	assert.True(t, unicode.In('_', gcpInner))
	assert.False(t, unicode.In('!', gcpInner))
}

func TestGcpNormalizeKey(t *testing.T) {
	assert.Equal(t, "test", gcpTag.NormalizeKey("test"))
	assert.Equal(t, "cafe", gcpTag.NormalizeKey("café 🍎?"))
	assert.Equal(t, "cafe_foo", gcpTag.NormalizeKey("__café_foo__"))
}

func TestGcpNormalizeValue(t *testing.T) {
	assert.Equal(t, "test", gcpTag.NormalizeValue("test"))
	assert.Equal(t, "cafe", gcpTag.NormalizeValue("café 🍎?"))
	assert.Equal(t, "cafe_foo", gcpTag.NormalizeValue("__café_foo__"))
}

func TestGcpValidateKey(t *testing.T) {
	assert.ErrorContains(t, gcpTag.ValidateKey(""), "not be empty")
	assert.ErrorContains(t, gcpTag.ValidateKey(strings.Repeat("a", 64)), "length")
	assert.ErrorContains(t, gcpTag.ValidateKey("café 🍎"), "latin")
	assert.ErrorContains(t, gcpTag.ValidateKey("????"), "pattern")
	assert.NoError(t, gcpTag.ValidateKey(strings.Repeat("a", 32)))
	assert.NoError(t, gcpTag.ValidateKey(gcpTag.NormalizeKey("café 🍎")))
}

func TestGcpValidateValue(t *testing.T) {
	assert.ErrorContains(t, gcpTag.ValidateValue(strings.Repeat("a", 64)), "length")
	assert.ErrorContains(t, gcpTag.ValidateValue("café 🍎"), "latin")
	assert.ErrorContains(t, gcpTag.ValidateValue("????"), "pattern")
	assert.NoError(t, gcpTag.ValidateValue(strings.Repeat("a", 32)))
	assert.NoError(t, gcpTag.ValidateValue(gcpTag.NormalizeValue("café 🍎")))
}
