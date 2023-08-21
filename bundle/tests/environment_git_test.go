package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitAutoLoadWithEnvironment(t *testing.T) {
	b := load(t, "./environments_autoload_git")
	assert.True(t, b.Config.Bundle.Git.Inferred)
	assert.Contains(t, b.Config.Bundle.Git.OriginURL, "/cli")
}

func TestGitManuallySetBranchWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_autoload_git", "production")
	assert.False(t, b.Config.Bundle.Git.Inferred)
	assert.Equal(t, "main", b.Config.Bundle.Git.Branch)
	assert.Contains(t, b.Config.Bundle.Git.OriginURL, "/cli")
}
