package textutil

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func TestLatinTable(t *testing.T) {
	assert.True(t, unicode.In('\u0000', Latin1))
	assert.True(t, unicode.In('A', Latin1))
	assert.True(t, unicode.In('Z', Latin1))
	assert.True(t, unicode.In('\u00ff', Latin1))
	assert.False(t, unicode.In('\u0100', Latin1))
}
