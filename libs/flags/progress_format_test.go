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
	assert.ErrorContains(t, err, "accepted arguments are [append]")

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
}
