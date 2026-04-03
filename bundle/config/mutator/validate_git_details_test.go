package mutator

import (
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
	diags := bundle.Apply(t.Context(), b, m)
	assert.NoError(t, diags.Error())
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
	diags := bundle.Apply(t.Context(), b, m)

	assert.ErrorContains(t, diags.Error(), "not on the right Git branch")
	assert.ErrorContains(t, diags.Error(), "expected according to configuration: main")
	assert.ErrorContains(t, diags.Error(), "actual: feature")
	assert.ErrorContains(t, diags.Error(), "Only use --force if you intentionally want to deploy from this branch")
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
	diags := bundle.Apply(t.Context(), b, m)
	assert.NoError(t, diags.Error())
}
