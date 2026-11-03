package dms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldsMask(t *testing.T) {
	// The order is fixed, so the same set always sends the same mask.
	assert.Equal(t, []string{"state", "error_message", "resource_id", "status"}, DescribesResource.Mask().Paths)
	assert.Equal(t, []string{"error_message", "status"}, KeepsState.Mask().Paths)
	assert.Equal(t, []string{"state"}, FieldState.Mask().Paths)
	assert.Empty(t, Fields(0).Mask().Paths)
}

func TestFieldsHas(t *testing.T) {
	assert.True(t, DescribesResource.Has(FieldState))
	assert.False(t, KeepsState.Has(FieldState))
	// Has asks for every field, not any of them.
	assert.True(t, DescribesResource.Has(FieldState|FieldStatus))
	assert.False(t, KeepsState.Has(FieldState|FieldStatus))
}
