package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/env"
	"github.com/stretchr/testify/assert"
)

func TestValidateValidateWorkspaceHost(t *testing.T) {
	host := "https://host.databricks.com"
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: host,
			},
		},
	}
	t.Setenv(env.HostVariable, host)

	m := ValidateWorkspaceHost()
	diags := bundle.Apply(context.Background(), b, m)
	assert.NoError(t, diags.Error())
}

func TestValidateValidateWorkspaceHostMismatch(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://target-host.databricks.com",
			},
		},
	}
	t.Setenv(env.HostVariable, "https://env-host.databricks.com")

	m := ValidateWorkspaceHost()
	diags := bundle.Apply(context.Background(), b, m)
	expectedError := "target host and DATABRICKS_HOST environment variable mismatch"
	assert.EqualError(t, diags.Error(), expectedError)
}
