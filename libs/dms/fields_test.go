package dms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldsMask(t *testing.T) {
	// The order is fixed, so the same set always sends the same mask.
	assert.Equal(t, "state,error_message,resource_id,status", DescribesResource.Mask())
	assert.Equal(t, "error_message,status", KeepsState.Mask())
	assert.Equal(t, "state", FieldState.Mask())
	assert.Empty(t, Fields(0).Mask())
}

func TestFieldsHas(t *testing.T) {
	assert.True(t, DescribesResource.Has(FieldState))
	assert.False(t, KeepsState.Has(FieldState))
	// Has asks for every field, not any of them.
	assert.True(t, DescribesResource.Has(FieldState|FieldStatus))
	assert.False(t, KeepsState.Has(FieldState|FieldStatus))
}
