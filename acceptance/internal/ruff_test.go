package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuffVersionOK(t *testing.T) {
	assert.True(t, ruffVersionOK("ruff 0.9.1", "0.9.1"))
	assert.True(t, ruffVersionOK("ruff 0.9.2", "0.9.1"))
	assert.True(t, ruffVersionOK("ruff 0.11.0", "0.9.1"))
	assert.False(t, ruffVersionOK("ruff 0.9.0", "0.9.1"))
	assert.False(t, ruffVersionOK("ruff 0.8.5", "0.9.1"))
}
