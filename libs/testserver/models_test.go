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

	// The rejected model must not be stored: that is the original bug, where a
	// deploy appeared to succeed and the next plan saw the resource as missing.
	// Checked before the require below so it is still reported when the rejection
	// is missing altogether.
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

	// Read the model back through the GET handler, so the test fails if create
	// stores it under a key the CLI cannot look up.
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
