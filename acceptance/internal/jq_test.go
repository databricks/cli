package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJqVersionOK(t *testing.T) {
	assert.True(t, jqVersionOK("jq-1.7", "1.7"))
	assert.True(t, jqVersionOK("jq-1.7.1", "1.7"))
	assert.True(t, jqVersionOK("jq-1.8.1", "1.7"))
	assert.False(t, jqVersionOK("jq-1.6", "1.7"))
	assert.False(t, jqVersionOK("garbage", "1.7"))
}
