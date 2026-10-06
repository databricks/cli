package shortid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	a := New()
	assert.Regexp(t, `^[0-9A-Za-z]{11}$`, a)
	assert.NotEqual(t, a, New())
}

func TestHash(t *testing.T) {
	// First 8 bytes of sha256("abc") are ba 78 16 bf 8f 01 cf ea.
	assert.Equal(t, "G0ZNUjK18Pa", Hash("abc"))
	assert.NotEqual(t, Hash("abc"), Hash("abd"))
}

func TestEncodeBounds(t *testing.T) {
	assert.Equal(t, "00000000000", encode([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
	assert.Equal(t, "LygHa16AHYF", encode([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}))
}
