package config_tests

import (
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
)

func TestGitAutoLoad(t *testing.T) {
	b := load(t, "./autoload_git")
	bundle.Apply(context.Background(), b, mutator.LoadGitDetails())
	validUrl := strings.Contains(b.Config.Bundle.Git.OriginURL, "/cli") || strings.Contains(b.Config.Bundle.Git.OriginURL, "/bricks")
	assert.True(t, validUrl, "Expected URL to contain '/cli' or '/bricks', got %s", b.Config.Bundle.Git.OriginURL)
}

func TestGitManuallySetBranch(t *testing.T) {
	b := loadTarget(t, "./autoload_git", "production")
	bundle.Apply(context.Background(), b, mutator.LoadGitDetails())
	assert.Equal(t, "main", b.Config.Bundle.Git.Branch)
	validUrl := strings.Contains(b.Config.Bundle.Git.OriginURL, "/cli") || strings.Contains(b.Config.Bundle.Git.OriginURL, "/bricks")
	assert.True(t, validUrl, "Expected URL to contain '/cli' or '/bricks', got %s", b.Config.Bundle.Git.OriginURL)
}

func TestGitBundleBranchValidation(t *testing.T) {
	git.GitDirectoryName = ".mock-git"
	t.Cleanup(func() {
		git.GitDirectoryName = ".git"
	})

	b := load(t, "./git_branch_validation")
	bundle.Apply(context.Background(), b, mutator.LoadGitDetails())
	assert.Equal(t, "feature-a", b.Config.Bundle.Git.Branch)
	assert.Equal(t, "feature-b", b.Config.Bundle.Git.ActualBranch)

	diags := bundle.Apply(context.Background(), b, mutator.ValidateGitDetails())
	assert.ErrorContains(t, diags.Error(), "not on the right Git branch:")
}
