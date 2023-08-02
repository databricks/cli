package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateIsInteger(t *testing.T) {
	assert.False(t, isIntegerValue(1.1))
	assert.False(t, isIntegerValue(0.1))
	assert.False(t, isIntegerValue(-0.1))

	assert.True(t, isIntegerValue(-1.0))
	assert.True(t, isIntegerValue(0.0))
	assert.True(t, isIntegerValue(2.0))
}
