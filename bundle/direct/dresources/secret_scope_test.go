package dresources_test

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
)

func TestPrepareStatePreservesEmptyBackendType(t *testing.T) {
	r := &dresources.ResourceSecretScope{}

	// When backend_type is not specified, PrepareState stores it as empty.
	// The backend_defaults rule in resources.yml prevents drift when the API
	// returns "DATABRICKS" as the default.
	input := &resources.SecretScope{
		Name: "test-scope",
	}
	state := r.PrepareState(input)
	assert.Equal(t, workspace.ScopeBackendType(""), state.ScopeBackendType)
}

func TestPrepareStatePreservesExplicitBackendType(t *testing.T) {
	r := &dresources.ResourceSecretScope{}

	tests := []struct {
		name        string
		backendType workspace.ScopeBackendType
	}{
		{"DATABRICKS", workspace.ScopeBackendTypeDatabricks},
		{"AZURE_KEYVAULT", workspace.ScopeBackendTypeAzureKeyvault},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &resources.SecretScope{
				Name:        "test-scope",
				BackendType: tt.backendType,
			}
			state := r.PrepareState(input)
			assert.Equal(t, tt.backendType, state.ScopeBackendType)
		})
	}
}
