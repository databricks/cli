package testserver

import (
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelRegistryCreateModel_RejectsEmptyName(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response, ok := workspace.ModelRegistryCreateModel(Request{Body: []byte(`{"name": ""}`)}).(Response)
	require.True(t, ok)
	assert.Equal(t, 400, response.StatusCode)

	// A stored-but-unreadable model is the original bug. Asserted before the
	// require below so it is still reported when the rejection is missing.
	assert.Empty(t, workspace.ModelRegistryModels)

	body, ok := response.Body.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "INVALID_PARAMETER_VALUE", body["error_code"])
	assert.Contains(t, body["message"], "cannot be empty strings")
}

func TestModelRegistryCreateModel_AllowsNonEmptyName(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response, ok := workspace.ModelRegistryCreateModel(Request{Body: []byte(`{"name": "my_model"}`)}).(Response)
	require.True(t, ok)
	// StatusCode 0 gets converted to 200 by normalizeResponse in the server
	require.Equal(t, 0, response.StatusCode)

	// Read back through the GET handler: a mis-keyed store must fail here.
	getResponse, ok := workspace.ModelRegistryGetModel(Request{
		URL: &url.URL{RawQuery: "name=my_model"},
	}).(Response)
	require.True(t, ok)
	require.Equal(t, 0, getResponse.StatusCode)

	body, ok := getResponse.Body.(ml.GetModelResponse)
	require.True(t, ok)
	assert.Equal(t, "my_model", body.RegisteredModelDatabricks.Name)
	assert.NotEmpty(t, body.RegisteredModelDatabricks.Id)
}
