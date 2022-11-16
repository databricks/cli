package bundle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadNotExists(t *testing.T) {
	b, err := Load("/doesntexist")
	assert.True(t, os.IsNotExist(err))
	assert.Nil(t, b)
}

func TestLoadExists(t *testing.T) {
	b, err := Load("./config/tests/basic")
	require.Nil(t, err)
	assert.Equal(t, "basic", b.Config.Bundle.Name)
}
