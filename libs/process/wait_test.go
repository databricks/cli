package process

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWait(t *testing.T) {
	err := Wait(1000000)
	assert.EqualError(t, err, "process with pid 1000000 does not exist")
}
