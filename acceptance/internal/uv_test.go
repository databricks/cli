package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUvVersionOK(t *testing.T) {
	assert.True(t, uvVersionOK("uv 0.4.0", "0.4"))
	assert.True(t, uvVersionOK("uv 0.11.22 (abcdef 2025-01-01)", "0.4"))
	assert.True(t, uvVersionOK("uv 1.0.0", "0.4"))
	assert.False(t, uvVersionOK("uv 0.3.5", "0.4"))
	assert.False(t, uvVersionOK("garbage", "0.4"))
}
