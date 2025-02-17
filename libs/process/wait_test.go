package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: Test this in windows. Setup an IDE.
func TestWait(t *testing.T) {
	err := Wait(1000000)
	assert.EqualError(t, err, "process with pid 1000000 does not exist")
}
