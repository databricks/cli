package filer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteMode(t *testing.T) {
	assert.Equal(t, 512, int(OverwriteIfExists))
	assert.Equal(t, 1024, int(CreateParentDirectories))
}
