package sync

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGithubGetPAT(t *testing.T) {
	pat, err := githubGetPAT(context.Background())
	assert.NoError(t, err)
	assert.NotEqual(t, "..", pat)
}