package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitConfig(t *testing.T) {
	b := load(t, "./autoload_git")
	assert.Equal(t, "foo", b.Config.Bundle.Git.Branch)
	assert.Equal(t, `https://github.com/databricks/bricks`, b.Config.Bundle.Git.RemoteUrl)
}
