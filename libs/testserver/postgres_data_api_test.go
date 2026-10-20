package testserver

import (
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDataApiBranch   = "projects/p/branches/production"
	testDataApiDatabase = testDataApiBranch + "/databases/db"
	testDataApiName     = testDataApiDatabase + "/data-api"
)

// dataApiWorkspace returns a workspace with a database on the project's root
// branch, which comes with a read-write endpoint.
func dataApiWorkspace(t *testing.T) *FakeWorkspace {
	workspace := NewFakeWorkspace("http://test.test", "token")
	resp := workspace.PostgresProjectCreate(Request{}, "p")
	require.Equal(t, 0, resp.StatusCode)
	resp = workspace.PostgresDatabaseCreate(Request{Body: []byte(`{"spec": {"postgres_database": "app_db", "role": "` + testDataApiBranch + `/roles/owner"}}`)}, testDataApiBranch, "db", false)
	require.Equal(t, 0, resp.StatusCode)
	return workspace
}

func updateDataApi(workspace *FakeWorkspace, mask, body string) Response {
	return workspace.PostgresDataApiUpdate(Request{
		URL:  &url.URL{RawQuery: url.Values{"update_mask": {mask}}.Encode()},
		Body: []byte(body),
	}, testDataApiName)
}

func TestPostgresDataApiCreateGetUpdateDelete(t *testing.T) {
	workspace := dataApiWorkspace(t)

	resp := workspace.PostgresDataApiCreate(Request{Body: []byte(`{"spec": {"db_schemas": ["public"], "db_max_rows": 100}}`)}, testDataApiDatabase)
	require.Equal(t, 0, resp.StatusCode)

	resp = workspace.PostgresDataApiGet(testDataApiName)
	require.Equal(t, 0, resp.StatusCode)
	dataApi := resp.Body.(postgres.DataApi)
	assert.Equal(t, testDataApiName, dataApi.Name)
	assert.Equal(t, testDataApiDatabase, dataApi.Parent)
	assert.Nil(t, dataApi.Spec)
	require.NotNil(t, dataApi.Status)
	assert.Equal(t, []string{"public"}, dataApi.Status.DbSchemas)
	assert.Equal(t, 100, dataApi.Status.DbMaxRows)

	// The "spec" mask replaces the whole configuration, so db_max_rows is reset.
	resp = updateDataApi(workspace, "spec", `{"spec": {"db_schemas": ["api"]}}`)
	require.Equal(t, 0, resp.StatusCode)
	dataApi = workspace.PostgresDataApis[testDataApiName]
	assert.Equal(t, []string{"api"}, dataApi.Status.DbSchemas)
	assert.Equal(t, 0, dataApi.Status.DbMaxRows)

	resp = workspace.PostgresDataApiDelete(testDataApiName)
	require.Equal(t, 0, resp.StatusCode)
	assert.Equal(t, 404, workspace.PostgresDataApiGet(testDataApiName).StatusCode)
	assert.Equal(t, 404, workspace.PostgresDataApiDelete(testDataApiName).StatusCode)
}

func TestPostgresDataApiCreateErrors(t *testing.T) {
	workspace := dataApiWorkspace(t)
	spec := Request{Body: []byte(`{"spec": {"db_schemas": ["public"]}}`)}

	resp := workspace.PostgresDataApiCreate(Request{Body: []byte(`{"spec": {}}`)}, testDataApiDatabase)
	assert.Equal(t, 400, resp.StatusCode)

	resp = workspace.PostgresDataApiCreate(spec, testDataApiBranch+"/databases/missing")
	assert.Equal(t, 404, resp.StatusCode)

	require.Equal(t, 0, workspace.PostgresDataApiCreate(spec, testDataApiDatabase).StatusCode)
	resp = workspace.PostgresDataApiCreate(spec, testDataApiDatabase)
	assert.Equal(t, 409, resp.StatusCode)
}

func TestPostgresDataApiRequiresReadWriteEndpoint(t *testing.T) {
	workspace := dataApiWorkspace(t)
	delete(workspace.PostgresEndpoints, testDataApiBranch+"/endpoints/primary")

	resp := workspace.PostgresDataApiCreate(Request{Body: []byte(`{"spec": {"db_schemas": ["public"]}}`)}, testDataApiDatabase)
	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, 400, workspace.PostgresDataApiGet(testDataApiName).StatusCode)
}

func TestPostgresDataApiUpdateRejectsLeafMask(t *testing.T) {
	workspace := dataApiWorkspace(t)
	require.Equal(t, 0, workspace.PostgresDataApiCreate(Request{Body: []byte(`{"spec": {"db_schemas": ["public"]}}`)}, testDataApiDatabase).StatusCode)

	resp := updateDataApi(workspace, "spec.db_max_rows", `{"spec": {"db_max_rows": 5}}`)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestPostgresDataApiGoneWithDatabase(t *testing.T) {
	workspace := dataApiWorkspace(t)
	require.Equal(t, 0, workspace.PostgresDataApiCreate(Request{Body: []byte(`{"spec": {"db_schemas": ["public"]}}`)}, testDataApiDatabase).StatusCode)

	require.Equal(t, 0, workspace.PostgresDatabaseDelete(testDataApiDatabase).StatusCode)
	assert.NotContains(t, workspace.PostgresDataApis, testDataApiName)
	assert.Equal(t, 404, workspace.PostgresDataApiGet(testDataApiName).StatusCode)
}
