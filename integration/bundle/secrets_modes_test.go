package bundle_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBundleSecretsWithTerraformMode tests secrets deployment using Terraform mode (default)
func TestBundleSecretsWithTerraformMode(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		t.Skip("Skipping on ucws environment")
	}

	ctx, wt := acc.WorkspaceTest(t)

	// Generate unique names for the test
	uniqueId := testutil.RandomName("secrets-tf-test-")
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
    test_secret:
      scope: %s
      key: test_key
      string_value: "terraform_mode_value"
`, uniqueId, scopeName, scopeName)

	err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Explicitly set Terraform mode (this is the default)
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENGINE", "terraform")

	// Deploy the bundle
	t.Log("Deploying bundle with Terraform mode...")
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve")
	_, _, err = c.Run()
	require.NoError(t, err)

	// Verify the secret exists and has correct value
	t.Log("Verifying secret...")
	assertSecretValue(t, wt, scopeName, "test_key", "terraform_mode_value")

	// Clean up
	t.Log("Destroying bundle...")
	destroyBundle(t, ctx, bundleRoot)
}

// TestBundleSecretsWithDirectMode tests secrets deployment using direct mode
func TestBundleSecretsWithDirectMode(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		t.Skip("Skipping on ucws environment")
	}

	ctx, wt := acc.WorkspaceTest(t)

	// Generate unique names for the test
	uniqueId := testutil.RandomName("secrets-direct-test-")
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
    test_secret:
      scope: %s
      key: test_key
      string_value: "direct_mode_value"
`, uniqueId, scopeName, scopeName)

	err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Set direct mode
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENGINE", "direct")

	// Deploy the bundle
	t.Log("Deploying bundle with direct mode...")
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve")
	_, _, err = c.Run()
	require.NoError(t, err)

	// Verify the secret exists and has correct value
	t.Log("Verifying secret...")
	assertSecretValue(t, wt, scopeName, "test_key", "direct_mode_value")

	// Clean up
	t.Log("Destroying bundle...")
	c2 := testcli.NewRunner(t, ctx, "bundle", "destroy", "--auto-approve")
	_, _, err = c2.Run()
	require.NoError(t, err)
}

// TestBundleSecretsSwitchModes tests switching between modes
func TestBundleSecretsSwitchModes(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		t.Skip("Skipping on ucws environment")
	}

	ctx, wt := acc.WorkspaceTest(t)

	// Generate unique names for the test
	uniqueId := testutil.RandomName("secrets-switch-test-")
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
    test_secret:
      scope: %s
      key: test_key
      string_value: "initial_value"
`, uniqueId, scopeName, scopeName)

	err := os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(bundleConfig), 0o644)
	require.NoError(t, err)

	// Deploy with Terraform mode first
	t.Log("Deploying with Terraform mode...")
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENGINE", "terraform")
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve")
	_, _, err = c.Run()
	require.NoError(t, err)

	// Verify the secret
	assertSecretValue(t, wt, scopeName, "test_key", "initial_value")

	// Update the config
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
    test_secret:
      scope: %s
      key: test_key
      string_value: "updated_value"
`, uniqueId, scopeName, scopeName)

	err = os.WriteFile(filepath.Join(bundleRoot, "databricks.yml"), []byte(updatedConfig), 0o644)
	require.NoError(t, err)

	// Deploy with direct mode
	t.Log("Deploying with direct mode...")
	ctx = env.Set(ctx, "DATABRICKS_BUNDLE_ENGINE", "direct")
	c2 := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve")
	_, _, err = c2.Run()

	// Note: Switching modes may or may not be supported depending on state management
	// If it's not supported, we should get an error
	if err != nil {
		t.Log("Mode switching not supported (expected):", err)
		assert.Contains(t, err.Error(), "state") // Should mention state issues
	} else {
		t.Log("Mode switching succeeded")
		// If it works, verify the value was updated
		assertSecretValue(t, wt, scopeName, "test_key", "updated_value")
	}

	// Clean up
	t.Log("Cleaning up...")
	// Use the same mode for cleanup
	c3 := testcli.NewRunner(t, ctx, "bundle", "destroy", "--auto-approve")
	_, _, _ = c3.Run() // Best effort cleanup
}
