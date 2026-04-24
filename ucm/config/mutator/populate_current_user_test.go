package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulateCurrentUser_SkipsWhenAlreadySet(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Workspace.Host = "https://example.cloud.databricks.com"
	preset := &config.User{User: &iam.User{UserName: "alice@example.com"}}
	u.CurrentUser = preset

	diags := ucm.Apply(t.Context(), u, mutator.PopulateCurrentUser())
	require.Empty(t, diags)
	// Identity-preserving: didn't rebuild or call the workspace client.
	assert.Same(t, preset, u.CurrentUser)
}

func TestPopulateCurrentUser_ErrorsWhenNoCredentials(t *testing.T) {
	u := &ucm.Ucm{Config: config.Root{}}
	u.Config.Workspace.Host = "https://no-such-host.example.invalid"

	diags := ucm.Apply(t.Context(), u, mutator.PopulateCurrentUser())
	require.NotEmpty(t, diags)
	// Don't pin the exact SDK error string — just that some auth diagnostic surfaced.
	assert.Nil(t, u.CurrentUser)
}
