package tnstate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert.True(t, CLIVersion{}.EqualORGreater(CLIVersion{}))
	assert.True(t, CLIVersion{1, 0, 0}.EqualORGreater(CLIVersion{1, 0, 0}))
	assert.False(t, CLIVersion{1, 0, 0}.EqualORGreater(CLIVersion{1, 0, 1}))
	assert.False(t, CLIVersion{1, 0, 0}.EqualORGreater(CLIVersion{1, 1, 0}))
	assert.True(t, CLIVersion{1, 0, 0}.EqualORGreater(CLIVersion{0, 999, 0}))
}
