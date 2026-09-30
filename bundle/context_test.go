package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOrNilReturnsNilWhenUnset(t *testing.T) {
	assert.Nil(t, GetOrNil(t.Context()))
}
