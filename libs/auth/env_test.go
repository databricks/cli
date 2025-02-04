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

func TestAuthEnvVars(t *testing.T) {
	expected := []string{
		"DATABRICKS_HOST",
		"DATABRICKS_CLUSTER_ID",
		"DATABRICKS_WAREHOUSE_ID",
		"DATABRICKS_SERVERLESS_COMPUTE_ID",
		"DATABRICKS_METADATA_SERVICE_URL",
		"DATABRICKS_ACCOUNT_ID",
		"DATABRICKS_TOKEN",
		"DATABRICKS_USERNAME",
		"DATABRICKS_PASSWORD",
		"DATABRICKS_CONFIG_PROFILE",
		"DATABRICKS_CONFIG_FILE",
		"DATABRICKS_GOOGLE_SERVICE_ACCOUNT",
		"GOOGLE_CREDENTIALS",
		"DATABRICKS_AZURE_RESOURCE_ID",
		"ARM_USE_MSI",
		"ARM_CLIENT_SECRET",
		"ARM_CLIENT_ID",
		"ARM_TENANT_ID",
		"ACTIONS_ID_TOKEN_REQUEST_URL",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN",
		"ARM_ENVIRONMENT",
		"DATABRICKS_AZURE_LOGIN_APP_ID",
		"DATABRICKS_CLIENT_ID",
		"DATABRICKS_CLIENT_SECRET",
		"DATABRICKS_CLI_PATH",
		"DATABRICKS_AUTH_TYPE",
		"DATABRICKS_DEBUG_TRUNCATE_BYTES",
		"DATABRICKS_DEBUG_HEADERS",
		"DATABRICKS_RATE_LIMIT",
	}

	out := EnvVars()
	assert.Equal(t, expected, out)
}
