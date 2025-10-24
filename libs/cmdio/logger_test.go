package cmdio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitAtLastNewLine(t *testing.T) {
	first, last := splitAtLastNewLine("hello\nworld")
	assert.Equal(t, "hello\n", first)
	assert.Equal(t, "world", last)

	first, last = splitAtLastNewLine("hello\r\nworld")
	assert.Equal(t, "hello\r\n", first)
	assert.Equal(t, "world", last)

	first, last = splitAtLastNewLine("hello world")
	assert.Equal(t, "", first)
	assert.Equal(t, "hello world", last)

	first, last = splitAtLastNewLine("hello\nworld\n")
	assert.Equal(t, "hello\nworld\n", first)
	assert.Equal(t, "", last)

	first, last = splitAtLastNewLine("\nhello world")
	assert.Equal(t, "\n", first)
	assert.Equal(t, "hello world", last)
}
