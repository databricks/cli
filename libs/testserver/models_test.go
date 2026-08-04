package testserver

import (
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
	assert.Equal(t, 0, response.StatusCode)

	body, ok := response.Body.(ml.CreateModelResponse)
	require.True(t, ok)
	assert.Equal(t, "my_model", body.RegisteredModel.Name)
}
