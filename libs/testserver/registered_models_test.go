package testserver

import (
	"testing"

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
