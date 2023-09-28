package tags

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func TestLatinTable(t *testing.T) {
	assert.True(t, unicode.In('\u0000', latin1))
	assert.True(t, unicode.In('A', latin1))
	assert.True(t, unicode.In('Z', latin1))
	assert.True(t, unicode.In('\u00ff', latin1))
	assert.False(t, unicode.In('\u0100', latin1))
}
