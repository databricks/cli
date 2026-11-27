package dresources

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A create request that times out lands after the first lookup: the repeated request is rejected
// with the 400 that Lakebase databases answer for a taken name, and the next lookup finds the
// database.
func TestPostgresDatabaseCreateRecoveryLooksAgainAfterAlreadyExists(t *testing.T) {
	const name = "projects/p/branches/b/databases/d"
	creates := []any{
		featureError(504, "DEADLINE_EXCEEDED", ""),
		featureError(400, "BAD_REQUEST", "database with that name already exists"),
	}
	gets := []any{
		featureError(404, "NOT_FOUND", "database not found"),
		postgres.Database{Name: name, CreateTime: sdktime.New(time.Now())},
	}
	createCount, getCount := 0, 0

	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/postgres/projects/p/branches/b/databases", func(req testserver.Request) any {
		createCount++
		require.LessOrEqual(t, createCount, len(creates), "unexpected create request")
		return creates[createCount-1]
	})
	server.Handle("GET", "/api/2.0/postgres/projects/p/branches/b/databases/d", func(req testserver.Request) any {
		getCount++
		require.LessOrEqual(t, getCount, len(gets), "unexpected get request")
		return gets[getCount-1]
	})
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	id, _, err := (*ResourcePostgresDatabase)(nil).New(client).DoCreate(ctx, &PostgresDatabaseState{DatabaseId: "d", Parent: "projects/p/branches/b"})
	require.NoError(t, err)
	assert.Equal(t, name, id)
	assert.Equal(t, 2, createCount)
	assert.Equal(t, 2, getCount)
}
