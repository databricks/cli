package bundle_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleSecretsDeployAndUpdate(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		t.Skip("Skipping on ucws environment")
	}

	ctx, wt := acc.WorkspaceTest(t)

	// Generate unique names for the test
	uniqueId := testutil.RandomName("secrets-test-")
	scopeName := fmt.Sprintf("%s-scope", uniqueId)

	// Create a temporary directory for the bundle
	bundleRoot := t.TempDir()

	// Write the bundle configuration
	bundleConfig := fmt.Sprintf(`
bundle:
  name: %s

workspace:
  root_path: ~/.bundle/${bundle.name}/${bundle.target}

resources:
  secret_scopes:
    test_scope:
      name: %s

  secrets:
    secret1:
      scope: %s
      key: secret_key_1
      string_value: "initial_value_1"

    secret2:
      scope: %s
      key: secret_key_2
      string_value: "initial_value_2"

    secret3:
      scope: %s
      key: secret_key_3
      string_value: "value_with_special_chars_!@#$%%"
`, uniqueId, scopeName, scopeName, scopeName, scopeName)

	err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Deploy the bundle
	t.Log("Deploying bundle with secrets...")
	deployBundle(t, ctx, bundleRoot)

	// Verify the secret scope was created
	t.Log("Verifying secret scope exists...")
	scopes, err := wt.W.Secrets.ListScopesAll(ctx)
	require.NoError(t, err)

	var foundScope bool
	for _, scope := range scopes {
		if scope.Name == scopeName {
			foundScope = true
			break
		}
	}
	assert.True(t, foundScope, "Secret scope should be created")

	// Verify secrets exist and have correct values
	t.Log("Verifying secrets exist and have correct values...")
	assertSecretValue(t, wt, scopeName, "secret_key_1", "initial_value_1")
	assertSecretValue(t, wt, scopeName, "secret_key_2", "initial_value_2")
	assertSecretValue(t, wt, scopeName, "secret_key_3", "value_with_special_chars_!@#$%")

	// Update one of the secrets
	t.Log("Updating secret value...")
	updatedConfig := fmt.Sprintf(`
bundle:
  name: %s

workspace:
  root_path: ~/.bundle/${bundle.name}/${bundle.target}

resources:
  secret_scopes:
    test_scope:
      name: %s

  secrets:
    secret1:
      scope: %s
      key: secret_key_1
      string_value: "updated_value_1"

    secret2:
      scope: %s
      key: secret_key_2
      string_value: "initial_value_2"

    secret3:
      scope: %s
      key: secret_key_3
      string_value: "value_with_special_chars_!@#$%%"
`, uniqueId, scopeName, scopeName, scopeName, scopeName)

	err = os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(updatedConfig), 0o644)
	require.NoError(t, err)

	// Deploy the updated bundle
	t.Log("Deploying updated bundle...")
	deployBundle(t, ctx, bundleRoot)

	// Verify the secret was updated
	t.Log("Verifying secret was updated...")
	assertSecretValue(t, wt, scopeName, "secret_key_1", "updated_value_1")
	assertSecretValue(t, wt, scopeName, "secret_key_2", "initial_value_2")

	// Clean up
	t.Log("Destroying bundle...")
	destroyBundle(t, ctx, bundleRoot)

	// Verify the scope was deleted
	t.Log("Verifying secret scope was deleted...")
	scopes, err = wt.W.Secrets.ListScopesAll(ctx)
	require.NoError(t, err)

	foundScope = false
	for _, scope := range scopes {
		if scope.Name == scopeName {
			foundScope = true
			break
		}
	}
	assert.False(t, foundScope, "Secret scope should be deleted after bundle destroy")
}

func TestBundleSecretsWithVariables(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		t.Skip("Skipping on ucws environment")
	}

	ctx, wt := acc.WorkspaceTest(t)

	// Generate unique names for the test
	uniqueId := testutil.RandomName("secrets-var-test-")
	scopeName := fmt.Sprintf("%s-scope", uniqueId)

	// Create a temporary directory for the bundle
	bundleRoot := t.TempDir()

	// Write the bundle configuration with variables
	bundleConfig := fmt.Sprintf(`
bundle:
  name: %s

workspace:
  root_path: ~/.bundle/${bundle.name}/${bundle.target}

variables:
  secret_value_1:
    description: "Secret value from variable"
    default: "variable_value_1"
  secret_value_2:
    description: "Secret value from variable"

resources:
  secret_scopes:
    test_scope:
      name: %s

  secrets:
    secret_from_var1:
      scope: %s
      key: var_secret_1
      string_value: ${var.secret_value_1}

    secret_from_var2:
      scope: %s
      key: var_secret_2
      string_value: ${var.secret_value_2}
`, uniqueId, scopeName, scopeName, scopeName)

	err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Deploy the bundle with a variable override for secret_value_2
	t.Log("Deploying bundle with variable-based secrets...")
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve", "--var", "secret_value_2=variable_value_2_from_flag")
	_, _, err = c.Run()
	require.NoError(t, err)

	// Verify secrets have the correct values from variables
	t.Log("Verifying secrets from variables...")
	assertSecretValue(t, wt, scopeName, "var_secret_1", "variable_value_1")
	assertSecretValue(t, wt, scopeName, "var_secret_2", "variable_value_2_from_flag")

	// Clean up
	t.Log("Destroying bundle...")
	destroyBundle(t, ctx, bundleRoot)
}

func TestBundleSecretsWithEmptyValue(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		t.Skip("Skipping on ucws environment")
	}

	ctx, wt := acc.WorkspaceTest(t)

	// Generate unique names for the test
	uniqueId := testutil.RandomName("secrets-empty-test-")
	scopeName := fmt.Sprintf("%s-scope", uniqueId)

	// Create a temporary directory for the bundle
	bundleRoot := t.TempDir()

	// Write the bundle configuration with an empty secret value
	bundleConfig := fmt.Sprintf(`
bundle:
  name: %s

workspace:
  root_path: ~/.bundle/${bundle.name}/${bundle.target}

resources:
  secret_scopes:
    test_scope:
      name: %s

  secrets:
    empty_secret:
      scope: %s
      key: empty_key
      string_value: ""
`, uniqueId, scopeName, scopeName)

	err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Deploy the bundle
	t.Log("Deploying bundle with empty secret...")
	deployBundle(t, ctx, bundleRoot)

	// Verify the secret exists and has an empty value
	t.Log("Verifying empty secret...")
	assertSecretValue(t, wt, scopeName, "empty_key", "")

	// Clean up
	t.Log("Destroying bundle...")
	destroyBundle(t, ctx, bundleRoot)
}

func TestBundleSecretsLifecycle(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		t.Skip("Skipping on ucws environment")
	}

	ctx, wt := acc.WorkspaceTest(t)

	// Generate unique names for the test
	uniqueId := testutil.RandomName("secrets-lifecycle-")
	scopeName := fmt.Sprintf("%s-scope", uniqueId)

	// Create a temporary directory for the bundle
	bundleRoot := t.TempDir()

	// Write the bundle configuration with a secret that prevents deletion
	bundleConfig := fmt.Sprintf(`
bundle:
  name: %s

workspace:
  root_path: ~/.bundle/${bundle.name}/${bundle.target}

resources:
  secret_scopes:
    test_scope:
      name: %s

  secrets:
    lifecycle_secret:
      scope: %s
      key: lifecycle_key
      string_value: "test_value"
      lifecycle:
        prevent_destroy: true
`, uniqueId, scopeName, scopeName)

	err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Deploy the bundle
	t.Log("Deploying bundle with lifecycle-protected secret...")
	deployBundle(t, ctx, bundleRoot)

	// Verify the secret exists
	assertSecretValue(t, wt, scopeName, "lifecycle_key", "test_value")

	// Update the bundle to remove the lifecycle protection
	bundleConfig = fmt.Sprintf(`
bundle:
  name: %s

workspace:
  root_path: ~/.bundle/${bundle.name}/${bundle.target}

resources:
  secret_scopes:
    test_scope:
      name: %s

  secrets:
    lifecycle_secret:
      scope: %s
      key: lifecycle_key
      string_value: "test_value"
`, uniqueId, scopeName, scopeName)

	err = os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Deploy the updated bundle
	t.Log("Deploying updated bundle without lifecycle protection...")
	deployBundle(t, ctx, bundleRoot)

	// Clean up
	t.Log("Destroying bundle...")
	destroyBundle(t, ctx, bundleRoot)
}

// assertSecretValue verifies that a secret exists and has the expected value
func assertSecretValue(t *testing.T, wt *acc.WorkspaceT, scope, key, expected string) {
	ctx := context.Background()

	// Get the secret metadata to verify it exists
	_, err := wt.W.Secrets.GetSecret(ctx, workspace.GetSecretRequest{
		Scope: scope,
		Key:   key,
	})
	require.NoError(t, err, "Secret %s/%s should exist", scope, key)

	// Use dbutils to get the actual secret value
	// We need to run Python code on a cluster to access dbutils.secrets
	out, err := wt.RunPython(fmt.Sprintf(`
		import base64
		value = dbutils.secrets.get(scope="%s", key="%s")
		encoded_value = base64.b64encode(value.encode('utf-8'))
		print(encoded_value.decode('utf-8'))
	`, scope, key))
	require.NoError(t, err, "Failed to retrieve secret value via dbutils")

	decoded, err := base64.StdEncoding.DecodeString(out)
	require.NoError(t, err, "Failed to decode secret value")
	assert.Equal(t, expected, string(decoded), "Secret %s/%s should have the expected value", scope, key)
}
