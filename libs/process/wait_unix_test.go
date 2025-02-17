package process

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWaitForPidUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on windows")
	}

	// Out of bounds pid. Should return an error.
	err := waitForPid(1000000)
	assert.EqualError(t, err, "process with pid 1000000 does not exist")
}
