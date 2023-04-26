package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToHttpsUrlForSsh(t *testing.T) {
	url, err := ToHttpsUrl("user@foo.com:org/repo-name.git")
	assert.NoError(t, err)
	assert.Equal(t, "https://foo.com/org/repo-name", url)
}
