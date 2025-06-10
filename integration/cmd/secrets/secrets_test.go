package secrets_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretsCreateScopeErrWhenNoArguments(t *testing.T) {
	ctx := context.Background()
	_, _, err := testcli.RequireErrorRun(t, ctx, "secrets", "create-scope")
	assert.Contains(t, err.Error(), "accepts 1 arg(s), received 0")
}

func temporarySecretScope(ctx context.Context, t *acc.WorkspaceT) string {
	scope := testutil.RandomName("cli-acc-")
	err := t.W.Secrets.CreateScope(ctx, workspace.CreateScope{
		Scope: scope,
	})
	require.NoError(t, err)

	// Delete the scope after the test.
	t.Cleanup(func() {
		err := t.W.Secrets.DeleteScopeByScope(ctx, scope)
		require.NoError(t, err)
	})

	return scope
}

func assertSecretStringValue(t *acc.WorkspaceT, scope, key, expected string) {
	out, err := t.RunPython(fmt.Sprintf(`
		import base64
		value = dbutils.secrets.get(scope="%s", key="%s")
		encoded_value = base64.b64encode(value.encode('utf-8'))
		print(encoded_value.decode('utf-8'))
	`, scope, key))
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(out)
	require.NoError(t, err)
	assert.Equal(t, expected, string(decoded))
}

func assertSecretBytesValue(t *acc.WorkspaceT, scope, key string, expected []byte) {
	out, err := t.RunPython(fmt.Sprintf(`
		import base64
		value = dbutils.secrets.getBytes(scope="%s", key="%s")
		encoded_value = base64.b64encode(value)
		print(encoded_value.decode('utf-8'))
	`, scope, key))
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(out)
	require.NoError(t, err)
	assert.Equal(t, expected, decoded)
}

func TestSecretsPutSecretStringValue(tt *testing.T) {
	// aws-prod-ucws sets CLOUD_ENV to "ucws"
	if os.Getenv("CLOUD_ENV") == "ucws" {
		/*
					FAIL integration/cmd/secrets.TestSecretsPutSecretStringValue (re-run 2) (1201.25s)
			      secrets_test.go:73:   args: secrets, put-secret, cli-acc-c4ea35e34b52466cbf9a090c431d6a0b, test-key, --string-value, test-value
			          with-newlines
			      secrets_test.go:40:
			          	Error Trace:	/home/runner/work/eng-dev-ecosystem/eng-dev-ecosystem/ext/cli/integration/internal/acc/workspace.go:75
			          	Error:      	Received unexpected error:
			          	            	wait: timed out: Finding instances for new nodes, acquiring more instances if necessary
			          	Messages:   	Unexpected error from EnsureClusterIsRunning for clusterID=***
		*/
		tt.Skip("Skipping to unblock PRs; re-enable if works")
	}

	ctx, t := acc.WorkspaceTest(tt)
	scope := temporarySecretScope(ctx, t)
	key := "test-key"
	value := "test-value\nwith-newlines\n"

	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "secrets", "put-secret", scope, key, "--string-value", value)
	assert.Empty(t, stdout)
	assert.Empty(t, stderr)

	assertSecretStringValue(t, scope, key, value)
	assertSecretBytesValue(t, scope, key, []byte(value))
}

func TestSecretsPutSecretBytesValue(tt *testing.T) {
	if os.Getenv("CLOUD_ENV") == "ucws" {
		tt.Skip("Skipping to unblock PRs; re-enable if works")
	}

	ctx, t := acc.WorkspaceTest(tt)
	scope := temporarySecretScope(ctx, t)
	key := "test-key"
	value := []byte{0x00, 0x01, 0x02, 0x03}

	stdout, stderr := testcli.RequireSuccessfulRun(t, ctx, "secrets", "put-secret", scope, key, "--bytes-value", string(value))
	assert.Empty(t, stdout)
	assert.Empty(t, stderr)

	// Note: this value cannot be represented as Python string,
	// so we only check equality through the dbutils.secrets.getBytes API.
	assertSecretBytesValue(t, scope, key, value)
}
