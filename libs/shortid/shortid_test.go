package shortid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	a := New()
	assert.Regexp(t, `^[0-9A-Za-z_-]{7}$`, a)
	assert.NotEqual(t, a, New())
}

func TestHash(t *testing.T) {
	// First 5 bytes of sha256("abc") are ba 78 16 bf 8f.
	assert.Equal(t, "ungWv48", Hash("abc"))
	assert.Equal(t, Hash("abc"), Hash("abc"))
	assert.NotEqual(t, Hash("abc"), Hash("abd"))
}
