package testserver

import (
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisteredModelsCreate_RejectsExistingModel(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	request := Request{Body: []byte(`{"catalog_name": "main", "schema_name": "default", "name": "mymodel", "comment": "first"}`)}

	require.Equal(t, 0, workspace.RegisteredModelsCreate(request).StatusCode)

	response := workspace.RegisteredModelsCreate(Request{Body: []byte(`{"catalog_name": "main", "schema_name": "default", "name": "mymodel", "comment": "second"}`)})
	assert.Equal(t, 400, response.StatusCode)
	assert.Equal(t, map[string]string{
		"error_code": "RESOURCE_ALREADY_EXISTS",
		"message":    "Routine or Model 'mymodel' already exists",
	}, response.Body)

	// The existing model is kept as it was.
	assert.Equal(t, "first", workspace.RegisteredModels["main.default.mymodel"].Comment)
}

func TestRegisteredModelsUpdate_RejectsRenameOntoExistingModel(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	for _, name := range []string{"model_a", "model_b"} {
		body := `{"catalog_name": "main", "schema_name": "default", "name": "` + name + `"}`
		require.Equal(t, 0, workspace.RegisteredModelsCreate(Request{Body: []byte(body)}).StatusCode)
	}

	response := workspace.RegisteredModelsUpdate(Request{Body: []byte(`{"new_name": "model_b"}`)}, "main.default.model_a")
	assert.Equal(t, 400, response.StatusCode)
	assert.Equal(t, map[string]string{
		"error_code": "RESOURCE_ALREADY_EXISTS",
		"message":    "Routine or Model 'model_b' already exists",
	}, response.Body)

	// Both models are kept as they were.
	assert.Contains(t, workspace.RegisteredModels, "main.default.model_a")
	assert.Contains(t, workspace.RegisteredModels, "main.default.model_b")
}

func TestRegisteredModels_NamesAreCaseInsensitive(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	require.Equal(t, 0, workspace.RegisteredModelsCreate(Request{Body: []byte(`{"catalog_name": "main", "schema_name": "default", "name": "MyModel"}`)}).StatusCode)

	// UC stores the name lowercased and finds the model under any casing.
	response := workspace.RegisteredModelsGet(Request{URL: &url.URL{}}, "main.default.MYMODEL")
	require.Equal(t, 0, response.StatusCode)
	model, ok := response.Body.(catalog.RegisteredModelInfo)
	require.True(t, ok)
	assert.Equal(t, "mymodel", model.Name)
	assert.Equal(t, "main.default.mymodel", model.FullName)

	// A rename that only changes the case keeps the lowercase name.
	response = workspace.RegisteredModelsUpdate(Request{Body: []byte(`{"new_name": "MyModel"}`)}, "main.default.mymodel")
	require.Equal(t, 0, response.StatusCode)
	model, ok = response.Body.(catalog.RegisteredModelInfo)
	require.True(t, ok)
	assert.Equal(t, "mymodel", model.Name)
}
