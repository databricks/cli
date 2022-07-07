package git

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGitOrigin(t *testing.T) {
	this, err := RepositoryName()
	assert.NoError(t, err)
	assert.Equal(t, "bricks", this)
}

func TestHttpsOrigin(t *testing.T) {
	url, err := HttpsOrigin()
	assert.NoError(t, err)
	// must pass on the upcoming forks
	assert.True(t, strings.HasPrefix(url, "https://github.com"), url)
	assert.True(t, strings.HasSuffix(url, "bricks.git"), url)
}