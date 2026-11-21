package cmdio

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The zero TestIOOptions must build the same cmdIO as NewTestIO, or the
// termtest harness would pin behaviour that no caller of NewTestIO sees.
func TestNewTestIOWithOptionsZeroValueMatchesNewTestIO(t *testing.T) {
	want := NewTestIO(nil, io.Discard, io.Discard)
	got := NewTestIOWithOptions(t.Context(), nil, io.Discard, io.Discard, TestIOOptions{})
	assert.Equal(t, want.capabilities, got.capabilities)
	assert.Equal(t, want.headerTemplate, got.headerTemplate)
	assert.Equal(t, want.template, got.template)
}
