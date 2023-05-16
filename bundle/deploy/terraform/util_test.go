package terraform

import (
	"fmt"
	"io"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

type mockedReader struct {
	content string
}

func (r *mockedReader) Read(p []byte) (n int, err error) {
	content := []byte(r.content)
	n = copy(p, content)
	return n, io.EOF
}

func TestLocalStateIsNewer(t *testing.T) {
	local := &mockedReader{content: `
{
	"serial": 5
}
`}
	remote := &mockedReader{content: `
{
	"serial": 4
}
`}

	stale := IsLocalStateStale(local, remote)

	assert.False(t, stale)
}

func TestLocalStateIsOlder(t *testing.T) {
	local := &mockedReader{content: `
{
	"serial": 5
}
`}
	remote := &mockedReader{content: `
{
	"serial": 6
}
`}

	stale := IsLocalStateStale(local, remote)
	assert.True(t, stale)
}

func TestLocalStateIsTheSame(t *testing.T) {
	local := &mockedReader{content: `
{
	"serial": 5
}
`}
	remote := &mockedReader{content: `
{
	"serial": 5
}
`}

	stale := IsLocalStateStale(local, remote)
	assert.False(t, stale)
}

func TestLocalStateMarkStaleWhenFailsToLoad(t *testing.T) {
	local := iotest.ErrReader(fmt.Errorf("Random error"))
	remote := &mockedReader{content: `
{
	"serial": 5
}
`}

	stale := IsLocalStateStale(local, remote)
	assert.True(t, stale)
}

func TestLocalStateMarkNonStaleWhenRemoteFailsToLoad(t *testing.T) {
	local := &mockedReader{content: `
{
	"serial": 5
}
`}
	remote := iotest.ErrReader(fmt.Errorf("Random error"))

	stale := IsLocalStateStale(local, remote)
	assert.False(t, stale)
}
