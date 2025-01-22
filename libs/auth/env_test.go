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
