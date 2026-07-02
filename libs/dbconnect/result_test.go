package dbconnect

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineErrorWrapsAndExposesCode(t *testing.T) {
	base := errors.New("boom")
	err := NewError(ErrFetch, base, "fetch %s", "x")
	assert.Equal(t, "fetch x: boom", err.Error())
	assert.Equal(t, ErrFetch, err.Code)
	assert.ErrorIs(t, err, base)
}

func TestModeString(t *testing.T) {
	assert.Equal(t, "default", ModeDefault.String())
	assert.Equal(t, "constraints-only", ModeConstraintsOnly.String())
}
