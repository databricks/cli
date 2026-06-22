package dbconnect

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineErrorWrapsAndExposesCode(t *testing.T) {
	base := errors.New("boom")
	err := NewError(ErrConstraintFetchFailed, base, "fetch %s", "x")
	assert.Equal(t, "fetch x: boom", err.Error())
	assert.Equal(t, ErrConstraintFetchFailed, err.Code)
	assert.ErrorIs(t, err, base)
}

func TestModeString(t *testing.T) {
	assert.Equal(t, "init", ModeInit.String())
	assert.Equal(t, "sync", ModeSync.String())
}
