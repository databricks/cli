package bundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundleInitIsRepoUrl(t *testing.T) {
	assert.True(t, isRepoUrl("git@github.com:databricks/cli.git"))
	assert.True(t, isRepoUrl("https://github.com/databricks/cli.git"))

	assert.False(t, isRepoUrl("./local"))
	assert.False(t, isRepoUrl("foo"))
}

func TestBundleInitRepoName(t *testing.T) {
	assert.Equal(t, repoName("git@github.com:databricks/cli.git"), "cli.git")
	assert.Equal(t, repoName("https://github.com/databricks/cli/"), "cli")
}
