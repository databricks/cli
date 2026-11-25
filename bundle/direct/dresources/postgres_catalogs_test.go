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

// createTestPostgresCatalog runs DoCreate against a server whose create request fails with 500
// while a catalog created at created exists.
func createTestPostgresCatalog(t *testing.T, created time.Time) (string, int, error) {
	t.Helper()
	creates := 0
	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/postgres/catalogs", func(req testserver.Request) any {
		creates++
		return branchError(500, "INTERNAL_ERROR", "The request could not be processed.")
	})
	server.Handle("GET", "/api/2.0/postgres/catalogs/{id}", func(req testserver.Request) any {
		return postgres.Catalog{Name: "catalogs/cat", CatalogId: "cat", CreateTime: sdktime.New(created)}
	})
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	id, _, err := (&ResourcePostgresCatalog{}).New(client).DoCreate(ctx, &PostgresCatalogState{CatalogId: "cat"})
	return id, creates, err
}

func TestPostgresCatalogCreateUsesCatalogCreatedDespiteError(t *testing.T) {
	id, creates, err := createTestPostgresCatalog(t, time.Now())
	require.NoError(t, err)
	assert.Equal(t, "catalogs/cat", id)
	assert.Equal(t, 1, creates)
}

func TestPostgresCatalogCreateRejectsCatalogThatPredatesRequest(t *testing.T) {
	_, creates, err := createTestPostgresCatalog(t, time.Now().Add(-time.Hour))
	require.ErrorContains(t, err, "catalog catalogs/cat exists but was not created by this deployment")
	requireStatus(t, err, 500)
	assert.Equal(t, 1, creates)
}
