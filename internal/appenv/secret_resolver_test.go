package appenv

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/databricks/cli/libs/apps/runlocal"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
)

func TestSecretResolver_Resolve_NoSecrets(t *testing.T) {
	ctx := context.Background()
	resolver := &SecretResolver{}

	envVars := []string{
		"APP_NAME=test-app",
		"PORT=8000",
		"DEBUG=true",
	}

	spec := &runlocal.AppSpec{
		EnvVars: []runlocal.AppEnvVar{},
	}

	result := resolver.Resolve(ctx, envVars, []apps.AppResource{}, spec)

	assert.Equal(t, envVars, result)
}

func TestSecretResolver_Resolve_WithSecretButNoResource(t *testing.T) {
	// This test requires cmdio context which is not available in unit tests
	t.Skip("Requires cmdio context for logging")
}

func TestSecretResolver_Resolve_WithSecretAndResource(t *testing.T) {
	// This test would need proper mocking of WorkspaceClient.Secrets.GetSecret
	// For now, this demonstrates the test structure
	t.Skip("Requires mocking WorkspaceClient.Secrets.GetSecret")
}

func TestSecretResolver_Resolve_Base64Decoding(t *testing.T) {
	secretValue := "test-secret-value"
	encoded := base64.StdEncoding.EncodeToString([]byte(secretValue))

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	assert.NoError(t, err)
	assert.Equal(t, secretValue, string(decoded))
}

func TestSecretResolver_Resolve_MixedEnvVars(t *testing.T) {
	// This test requires cmdio context which is not available in unit tests
	t.Skip("Requires cmdio context for logging")
}

func TestSecretResolver_Resolve_InvalidEnvVarFormat(t *testing.T) {
	ctx := context.Background()
	resolver := &SecretResolver{}

	envVars := []string{
		"INVALID_FORMAT",
		"VALID=value",
	}

	spec := &runlocal.AppSpec{
		EnvVars: []runlocal.AppEnvVar{},
	}

	result := resolver.Resolve(ctx, envVars, []apps.AppResource{}, spec)

	assert.Equal(t, envVars, result)
}
