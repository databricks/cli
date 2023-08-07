package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
)

func TestGitAutoLoad(t *testing.T) {
	b := load(t, "./autoload_git")
	assert.True(t, b.Config.Bundle.Git.Inferred)
	assert.Contains(t, b.Config.Bundle.Git.OriginURL, "/cli")
}

func TestGitManuallySetBranch(t *testing.T) {
	b := loadEnvironment(t, "./autoload_git", "production")
	assert.False(t, b.Config.Bundle.Git.Inferred)
	assert.Equal(t, "main", b.Config.Bundle.Git.Branch)
	assert.Contains(t, b.Config.Bundle.Git.OriginURL, "/cli")
}

func TestGitBundleBranchValidation(t *testing.T) {
	git.GitDirectoryName = ".mock-git"
	t.Cleanup(func() {
		git.GitDirectoryName = ".git"
	})

	b := load(t, "./git_branch_validation")
	assert.False(t, b.Config.Bundle.Git.Inferred)
	assert.Equal(t, "feature-a", b.Config.Bundle.Git.Branch)
	assert.Equal(t, "feature-b", b.Config.Bundle.Git.ActualBranch)

	err := bundle.Apply(context.Background(), b, mutator.ValidateGitDetails())
	assert.ErrorContains(t, err, "not on the right Git branch:")
}
