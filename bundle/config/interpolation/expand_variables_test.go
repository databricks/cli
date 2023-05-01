package interpolation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariableInterpolationIsVariableReference(t *testing.T) {
	assert.True(t, isVariableReference("var.foo"))
	assert.True(t, isVariableReference("var.bar"))
	assert.True(t, isVariableReference("var.var"))
	assert.True(t, isVariableReference("var.a-b-c"))

	assert.False(t, isVariableReference("var.foo.bar"))
	assert.False(t, isVariableReference("vars.bar"))
	assert.False(t, isVariableReference("var"))
	assert.False(t, isVariableReference("var."))
}

func TestVariableInterpolationExpandVariable(t *testing.T) {
	ans, err := expandVariable("var.foo")
	assert.NoError(t, err)
	assert.Equal(t, "variables.foo.value", ans)

	ans, err = expandVariable("var.foo-BAR")
	assert.NoError(t, err)
	assert.Equal(t, "variables.foo-BAR.value", ans)

	_, err = expandVariable("var.foo.bar")
	assert.ErrorContains(t, err, "var.foo.bar is not a valid variable reference")

	_, err = expandVariable("var.")
	assert.ErrorContains(t, err, "var. is not a valid variable reference")

	_, err = expandVariable("vars.foo")
	assert.ErrorContains(t, err, "vars.foo is not a valid variable reference")
}
