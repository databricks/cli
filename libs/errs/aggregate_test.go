package errs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromManyErrors(t *testing.T) {
	e1 := errors.New("Error 1")
	e2 := errors.New("Error 2")
	e3 := errors.New("Error 3")
	err := FromMany(e1, e2, e3)

	assert.ErrorIs(t, err, e1)
	assert.ErrorIs(t, err, e2)
	assert.ErrorIs(t, err, e3)

	assert.Equal(t, `Error 1
Error 2
Error 3`, err.Error())
}

func TestFromManyErrorsWihtNil(t *testing.T) {
	e1 := errors.New("Error 1")
	var e2 error = nil
	e3 := errors.New("Error 3")
	err := FromMany(e1, e2, e3)

	assert.ErrorIs(t, err, e1)
	assert.ErrorIs(t, err, e3)

	assert.Equal(t, `Error 1
Error 3`, err.Error())
}
