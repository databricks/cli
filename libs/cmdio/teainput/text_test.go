package teainput_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio/teainput"
	"github.com/stretchr/testify/assert"
)

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "item-1", teainput.FirstLine("item-1"))
	assert.Equal(t, "item-1", teainput.FirstLine("item-1\n"))
	assert.Equal(t, "item-1", teainput.FirstLine("item-1\r\nitem-2"))
	assert.Empty(t, teainput.FirstLine("\nitem-2"))
}
