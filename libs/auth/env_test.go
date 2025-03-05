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
		// Generic attributes.
		{"host", "DATABRICKS_HOST"},
		{"profile", "DATABRICKS_CONFIG_PROFILE"},
		{"auth_type", "DATABRICKS_AUTH_TYPE"},
		{"metadata_service_url", "DATABRICKS_METADATA_SERVICE_URL"},

		// OAuth specific attributes.
		{"client_id", "DATABRICKS_CLIENT_ID"},

		// Google specific attributes.
		{"google_service_account", "DATABRICKS_GOOGLE_SERVICE_ACCOUNT"},

		// Azure specific attributes.
		{"azure_workspace_resource_id", "DATABRICKS_AZURE_RESOURCE_ID"},
		{"azure_use_msi", "ARM_USE_MSI"},
		{"azure_client_id", "ARM_CLIENT_ID"},
		{"azure_tenant_id", "ARM_TENANT_ID"},
		{"azure_environment", "ARM_ENVIRONMENT"},
		{"azure_login_app_id", "DATABRICKS_AZURE_LOGIN_APP_ID"},
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

func TestAuthEnvVars(t *testing.T) {
	// Few common environment variables that we expect the SDK to support.
	contains := []string{
		// Generic attributes.
		"DATABRICKS_HOST",
		"DATABRICKS_CONFIG_PROFILE",
		"DATABRICKS_AUTH_TYPE",
		"DATABRICKS_METADATA_SERVICE_URL",
		"DATABRICKS_CONFIG_FILE",

		// OAuth specific attributes.
		"DATABRICKS_CLIENT_ID",
		"DATABRICKS_CLIENT_SECRET",
		"DATABRICKS_CLI_PATH",

		// Google specific attributes.
		"DATABRICKS_GOOGLE_SERVICE_ACCOUNT",
		"GOOGLE_CREDENTIALS",

		// Personal access token specific attributes.
		"DATABRICKS_TOKEN",

		// Databricks password specific attributes.
		"DATABRICKS_USERNAME",
		"DATABRICKS_PASSWORD",

		// Account authentication attributes.
		"DATABRICKS_ACCOUNT_ID",

		// Azure attributes
		"DATABRICKS_AZURE_RESOURCE_ID",
		"ARM_USE_MSI",
		"ARM_CLIENT_SECRET",
		"ARM_CLIENT_ID",
		"ARM_TENANT_ID",
		"ARM_ENVIRONMENT",

		// Github attributes
		"ACTIONS_ID_TOKEN_REQUEST_URL",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN",
	}

	out := envVars()
	for _, v := range contains {
		assert.Contains(t, out, v)
	}
}
