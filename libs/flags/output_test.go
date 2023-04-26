package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOutputFlag(t *testing.T) {
	var f Output
	var err error

	// Invalid
	err = f.Set("foo")
	assert.EqualError(t, err, "accepted arguments are json and text")

	// Lowercase
	err = f.Set("text")
	assert.NoError(t, err)
	assert.Equal(t, "text", f.String())

	// Uppercase
	err = f.Set("TEXT")
	assert.NoError(t, err)
	assert.Equal(t, "text", f.String())

	// Lowercase
	err = f.Set("json")
	assert.NoError(t, err)
	assert.Equal(t, "json", f.String())

	// Uppercase
	err = f.Set("JSON")
	assert.NoError(t, err)
	assert.Equal(t, "json", f.String())
}
