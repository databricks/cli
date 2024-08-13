package config_tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitAutoLoadWithEnvironment(t *testing.T) {
	b := load(t, "./environments_autoload_git")
	assert.True(t, b.Config.Bundle.Git.Inferred)
	validUrl := strings.Contains(b.Config.Bundle.Git.OriginURL, "/cli") || strings.Contains(b.Config.Bundle.Git.OriginURL, "/bricks")
	assert.True(t, validUrl, fmt.Sprintf("Expected URL to contain '/cli' or '/bricks', got %s", b.Config.Bundle.Git.OriginURL))
}

func TestGitManuallySetBranchWithEnvironment(t *testing.T) {
	b := loadTarget(t, "./environments_autoload_git", "production")
	assert.False(t, b.Config.Bundle.Git.Inferred)
	assert.Equal(t, "main", b.Config.Bundle.Git.Branch)
	validUrl := strings.Contains(b.Config.Bundle.Git.OriginURL, "/cli") || strings.Contains(b.Config.Bundle.Git.OriginURL, "/bricks")
	assert.True(t, validUrl, fmt.Sprintf("Expected URL to contain '/cli' or '/bricks', got %s", b.Config.Bundle.Git.OriginURL))
}
