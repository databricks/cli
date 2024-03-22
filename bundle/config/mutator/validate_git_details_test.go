package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateGitDetailsMatchingBranches(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Git: config.Git{
					Branch:       "main",
					ActualBranch: "main",
				},
			},
		},
	}

	m := ValidateGitDetails()
	diags := bundle.Apply(context.Background(), b, m)
	assert.Empty(t, diags)

}

func TestValidateGitDetailsNonMatchingBranches(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Git: config.Git{
					Branch:       "main",
					ActualBranch: "feature",
				},
			},
		},
	}

	m := ValidateGitDetails()
	err := bundle.Apply(context.Background(), b, m)

	expectedError := "not on the right Git branch:\n  expected according to configuration: main\n  actual: feature\nuse --force to override"
	assert.EqualError(t, err, expectedError)
}

func TestValidateGitDetailsNotUsingGit(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Git: config.Git{
					Branch:       "main",
					ActualBranch: "",
				},
			},
		},
	}

	m := ValidateGitDetails()
	diags := bundle.Apply(context.Background(), b, m)
	assert.Empty(t, diags)

}
