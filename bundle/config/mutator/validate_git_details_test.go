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

	err := diags.Error()
	assert.ErrorContains(t, err, "not on the right Git branch:\n  expected according to configuration: main\n  actual: feature")
	assert.ErrorContains(t, err, "To deploy from this branch anyway, use --force. Note that this may push unexpected code to the target.")
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
