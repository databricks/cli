package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRoleStateRoundTrip(t *testing.T) {
	state := PostgresRoleState{
		RoleId: "app-owner",
		Parent: "projects/p/branches/main",
		RoleRoleSpec: postgres.RoleRoleSpec{
			PostgresRole: "app_owner",
		},
	}

	b, err := json.Marshal(state)
	require.NoError(t, err)

	// role_id and parent must survive marshaling. The embedded RoleRoleSpec has
	// its own MarshalJSON that would otherwise take over and drop them from the
	// persisted state.
	var raw map[string]any
	require.NoError(t, json.Unmarshal(b, &raw))
	assert.Equal(t, "app-owner", raw["role_id"])
	assert.Equal(t, "projects/p/branches/main", raw["parent"])

	var got PostgresRoleState
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "app-owner", got.RoleId)
	assert.Equal(t, "projects/p/branches/main", got.Parent)
	assert.Equal(t, "app_owner", got.PostgresRole)
}
