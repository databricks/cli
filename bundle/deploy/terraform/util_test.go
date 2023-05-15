package terraform

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockedReader struct {
	content string
	failed  bool
}

func (r *mockedReader) Read(p []byte) (n int, err error) {
	if r.failed {
		return 0, fmt.Errorf("Failed to load")
	}
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
	local := &mockedReader{content: `
{
	failed: true
}
`}
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
	remote := &mockedReader{content: `
{
	failed: true
}
`}

	stale := IsLocalStateStale(local, remote)
	assert.False(t, stale)
}
