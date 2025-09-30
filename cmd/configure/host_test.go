package configure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateHost(t *testing.T) {
	var err error

	// Must start with https://
	err = validateHost("/path")
	assert.ErrorContains(t, err, "must start with https://")
	err = validateHost("http://host")
	assert.ErrorContains(t, err, "must start with https://")
	err = validateHost("ftp://host")

	// Must use empty path
	assert.ErrorContains(t, err, "must start with https://")
	err = validateHost("https://host/path")
	assert.ErrorContains(t, err, "must use empty path")

	// Ignore query params
	err = validateHost("https://host/?query")
	assert.NoError(t, err)
	err = validateHost("https://host/")
	assert.NoError(t, err)
}
