package auth

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestAuthEnv(t *testing.T) {
	in := &config.Config{
		Profile:            "thisshouldbeignored",
		Host:               "https://test.com",
		Token:              "test-token",
		Password:           "test-password",
		MetadataServiceURL: "http://somurl.com",

		AzureUseMSI:       true,
		AzureTenantID:     "test-tenant-id",
		AzureClientID:     "test-client-id",
		AzureClientSecret: "test-client-secret",

		ActionsIDTokenRequestToken: "test-actions-id-token-request-token",
	}

	expected := map[string]string{
		"DATABRICKS_HOST":                 "https://test.com",
		"DATABRICKS_TOKEN":                "test-token",
		"DATABRICKS_PASSWORD":             "test-password",
		"DATABRICKS_METADATA_SERVICE_URL": "http://somurl.com",

		"ARM_USE_MSI":       "true",
		"ARM_TENANT_ID":     "test-tenant-id",
		"ARM_CLIENT_ID":     "test-client-id",
		"ARM_CLIENT_SECRET": "test-client-secret",

		"ACTIONS_ID_TOKEN_REQUEST_TOKEN": "test-actions-id-token-request-token",
	}

	out := Env(in)
	assert.Equal(t, expected, out)
}

func TestGetEnvFor(t *testing.T) {
	tcases := []struct {
		name     string
		expected string
	}{
		{
			name:     "host",
			expected: "DATABRICKS_HOST",
		},
		{
			name:     "profile",
			expected: "DATABRICKS_CONFIG_PROFILE",
		},
		{
			name:     "auth_type",
			expected: "DATABRICKS_AUTH_TYPE",
		},
		{
			name:     "metadata_service_url",
			expected: "DATABRICKS_METADATA_SERVICE_URL",
		},
		{
			name:     "client_id",
			expected: "DATABRICKS_CLIENT_ID",
		},
		{
			name:     "google_service_account",
			expected: "DATABRICKS_GOOGLE_SERVICE_ACCOUNT",
		},
		{
			name:     "azure_workspace_resource_id",
			expected: "DATABRICKS_AZURE_RESOURCE_ID",
		},
		{
			name:     "azure_use_msi",
			expected: "ARM_USE_MSI",
		},
		{
			name:     "azure_client_id",
			expected: "ARM_CLIENT_ID",
		},
		{
			name:     "azure_tenant_id",
			expected: "ARM_TENANT_ID",
		},
		{
			name:     "azure_environment",
			expected: "ARM_ENVIRONMENT",
		},
		{
			name:     "azure_login_app_id",
			expected: "DATABRICKS_AZURE_LOGIN_APP_ID",
		},
	}

	for _, tcase := range tcases {
		t.Run(tcase.name, func(t *testing.T) {
			out, ok := GetEnvFor(tcase.name)
			assert.True(t, ok)
			assert.Equal(t, tcase.expected, out)
		})
	}

	out, ok := GetEnvFor("notfound")
	assert.False(t, ok)
	assert.Empty(t, out)
}
