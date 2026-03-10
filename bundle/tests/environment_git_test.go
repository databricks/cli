package config_tests

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestGitAutoLoadWithEnvironment(t *testing.T) {
	b := load(t, "./environments_autoload_git")
	bundle.Apply(t.Context(), b, mutator.LoadGitDetails())
	validUrl := strings.Contains(b.Config.Bundle.Git.OriginURL, "/cli") || strings.Contains(b.Config.Bundle.Git.OriginURL, "/bricks")
	assert.True(t, validUrl, "Expected URL to contain '/cli' or '/bricks', got %s", b.Config.Bundle.Git.OriginURL)
}

func TestGitManuallySetBranchWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_autoload_git", "production")
	bundle.Apply(t.Context(), b, mutator.LoadGitDetails())
	assert.Equal(t, "main", b.Config.Bundle.Git.Branch)
	validUrl := strings.Contains(b.Config.Bundle.Git.OriginURL, "/cli") || strings.Contains(b.Config.Bundle.Git.OriginURL, "/bricks")
	assert.True(t, validUrl, "Expected URL to contain '/cli' or '/bricks', got %s", b.Config.Bundle.Git.OriginURL)
}
