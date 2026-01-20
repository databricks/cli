package pipelines

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLooksLikeUUID(t *testing.T) {
	assert.True(t, looksLikeUUID("a12cd3e4-0ab1-1abc-1a2b-1a2bcd3e4f05"))
}

func TestLooksLikeUUID_resourceName(t *testing.T) {
	assert.False(t, looksLikeUUID("my-pipeline-key"))
}
