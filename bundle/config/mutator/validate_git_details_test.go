package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestValidateGitDetailsMatchingBranches(t *testing.T) {
	bundle := &bundle.Bundle{
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
	err := m.Apply(context.Background(), bundle)

	assert.NoError(t, err)
}

func TestValidateGitDetailsNonMatchingBranches(t *testing.T) {
	bundle := &bundle.Bundle{
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
	err := m.Apply(context.Background(), bundle)

	expectedError := "not on the right Git branch:\n  expected according to configuration: main\n  actual: feature"
	assert.EqualError(t, err, expectedError)
}

func TestValidateGitDetailsNotUsingGit(t *testing.T) {
	bundle := &bundle.Bundle{
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
	err := m.Apply(context.Background(), bundle)

	assert.NoError(t, err)
}
