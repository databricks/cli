package terraform

import (
	"fmt"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

func TestLocalStateIsNewer(t *testing.T) {
	local := strings.NewReader(`{"serial": 5}`)
	remote := strings.NewReader(`{"serial": 4}`)
	assert.False(t, IsLocalStateStale(local, remote))
}

func TestLocalStateIsOlder(t *testing.T) {
	local := strings.NewReader(`{"serial": 5}`)
	remote := strings.NewReader(`{"serial": 6}`)
	assert.True(t, IsLocalStateStale(local, remote))
}

func TestLocalStateIsTheSame(t *testing.T) {
	local := strings.NewReader(`{"serial": 5}`)
	remote := strings.NewReader(`{"serial": 5}`)
	assert.False(t, IsLocalStateStale(local, remote))
}

func TestLocalStateMarkStaleWhenFailsToLoad(t *testing.T) {
	local := iotest.ErrReader(fmt.Errorf("Random error"))
	remote := strings.NewReader(`{"serial": 5}`)
	assert.True(t, IsLocalStateStale(local, remote))
}

func TestLocalStateMarkNonStaleWhenRemoteFailsToLoad(t *testing.T) {
	local := strings.NewReader(`{"serial": 5}`)
	remote := iotest.ErrReader(fmt.Errorf("Random error"))
	assert.False(t, IsLocalStateStale(local, remote))
}
