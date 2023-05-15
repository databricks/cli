package errs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromManyErrors(t *testing.T) {
	e1 := fmt.Errorf("Error 1")
	e2 := fmt.Errorf("Error 2")
	e3 := fmt.Errorf("Error 3")
	err := FromMany(e1, e2, e3)

	assert.True(t, errors.Is(err, e1))
	assert.True(t, errors.Is(err, e2))
	assert.True(t, errors.Is(err, e3))

	assert.Equal(t, err.Error(), `Error 1
Error 2
Error 3`)
}

func TestFromManyErrorsWihtNil(t *testing.T) {
	e1 := fmt.Errorf("Error 1")
	var e2 error = nil
	e3 := fmt.Errorf("Error 3")
	err := FromMany(e1, e2, e3)

	assert.True(t, errors.Is(err, e1))
	assert.True(t, errors.Is(err, e3))

	assert.Equal(t, err.Error(), `Error 1
Error 3`)
}
