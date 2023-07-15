package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAutoLoad(t *testing.T) {
	b := load(t, "./autoload_git")
	assert.NotEqual(t, "", b.Config.Bundle.Git.Branch)
	assert.Contains(t, b.Config.Bundle.Git.OriginURL, "/cli")
}

func TestWrongBranch(t *testing.T) {
	err := loadEnvironmentWithError(t, "./autoload_git", "production")
	assert.ErrorContains(t, err, "not on the right Git branch")
}
