package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgressFormatNonTtyDefault(t *testing.T) {
	format := NewProgressLogFormat()
	assert.Equal(t, format, ModeDefault)
}

func TestProgressFormatSet(t *testing.T) {
	p := NewProgressLogFormat()

	// invalid arg
	err := p.Set("foo")
	assert.ErrorContains(t, err, "accepted arguments are [append, inplace, json]")

	// set json
	err = p.Set("json")
	assert.NoError(t, err)
	assert.Equal(t, "json", p.String())

	err = p.Set("JSON")
	assert.NoError(t, err)
	assert.Equal(t, "json", p.String())

	err = p.Set("Json")
	assert.NoError(t, err)
	assert.Equal(t, "json", p.String())

	// set append
	err = p.Set("append")
	assert.NoError(t, err)
	assert.Equal(t, "append", p.String())

	err = p.Set("Append")
	assert.NoError(t, err)
	assert.Equal(t, "append", p.String())

	err = p.Set("APPEND")
	assert.NoError(t, err)
	assert.Equal(t, "append", p.String())

	// set inplace
	err = p.Set("inplace")
	assert.NoError(t, err)
	assert.Equal(t, "inplace", p.String())

	err = p.Set("Inplace")
	assert.NoError(t, err)
	assert.Equal(t, "inplace", p.String())

	err = p.Set("INPLACE")
	assert.NoError(t, err)
	assert.Equal(t, "inplace", p.String())
}
