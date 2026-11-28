package dresources

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDatabaseCatalogName = "my_catalog"

// createTestDatabaseCatalog runs DoCreate against a server whose create request times out after
// the catalog exists, with the UC catalog reporting ucCreatedAt (milliseconds) as its create time.
// A negative ucCreatedAt makes the UC lookup fail with 404.
func createTestDatabaseCatalog(t *testing.T, ucCreatedAt int64) (string, int, error) {
	t.Helper()
	dbCatalog := database.DatabaseCatalog{Name: testDatabaseCatalogName, DatabaseInstanceName: "inst", DatabaseName: "db"}
	creates := 0
	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/database/catalogs", func(req testserver.Request) any {
		creates++
		return featureError(504, "DEADLINE_EXCEEDED", "")
	})
	server.Handle("GET", "/api/2.0/database/catalogs/{name}", func(req testserver.Request) any {
		return dbCatalog
	})
	server.Handle("GET", "/api/2.1/unity-catalog/catalogs/{name}", func(req testserver.Request) any {
		if ucCreatedAt < 0 {
			return featureError(404, "CATALOG_DOES_NOT_EXIST", "Catalog 'my_catalog' does not exist.")
		}
		return catalog.CatalogInfo{Name: testDatabaseCatalogName, CreatedAt: ucCreatedAt, CreatedBy: testserver.TestUser.UserName}
	})
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	id, _, err := (&ResourceDatabaseCatalog{}).New(client).DoCreate(ctx, &dbCatalog)
	return id, creates, err
}

func TestDatabaseCatalogCreateUsesCatalogCreatedDespiteTimeout(t *testing.T) {
	id, creates, err := createTestDatabaseCatalog(t, time.Now().UnixMilli())
	require.NoError(t, err)
	assert.Equal(t, testDatabaseCatalogName, id)
	assert.Equal(t, 1, creates)
}

func TestDatabaseCatalogCreateRejectsCatalogThatPredatesRequest(t *testing.T) {
	_, creates, err := createTestDatabaseCatalog(t, time.Now().Add(-time.Hour).UnixMilli())
	require.EqualError(t, err, "database catalog my_catalog exists but was not created by this deployment")
	requireStatus(t, err, 504)
	assert.Equal(t, 1, creates)
}

func TestDatabaseCatalogCreateRejectsCatalogWithoutCreateTime(t *testing.T) {
	_, _, err := createTestDatabaseCatalog(t, 0)
	require.EqualError(t, err, "database catalog my_catalog exists but has no create time, so it is not known to be created by this deployment")
}

func TestDatabaseCatalogCreateDoesNotRepeatCreateWhenOnlyUCLookupMisses(t *testing.T) {
	_, creates, err := createTestDatabaseCatalog(t, -1)
	require.EqualError(t, err, "database catalog my_catalog exists but has no create time, so it is not known to be created by this deployment")
	assert.Equal(t, 1, creates)
}
