package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuffVersionOK(t *testing.T) {
	assert.True(t, ruffVersionOK("ruff 0.9.1"))
	assert.True(t, ruffVersionOK("ruff 0.9.2"))
	assert.True(t, ruffVersionOK("ruff 0.11.0"))
	assert.True(t, ruffVersionOK("ruff 1.0.0"))
	assert.False(t, ruffVersionOK("ruff 0.9.0"))
	assert.False(t, ruffVersionOK("ruff 0.8.5"))
	assert.False(t, ruffVersionOK(""))
}
