package keys

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
	"github.com/databricks/sdk-go/secrets/v1"
)

// CreateKeysSecretScope creates or retrieves the secret scope for SSH keys.
// sessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
func CreateKeysSecretScope(ctx context.Context, client *sdkclients.Clients, sessionID string) (string, error) {
	me, err := client.Scim.Me(ctx, sdkscim.MeRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	secretScopeName := fmt.Sprintf("%s-%s-ssh-tunnel-keys", utils.Deref(me.UserName), sessionID)

	// Do not create the scope if it already exists.
	// We can instead filter out "resource already exists" errors from CreateScope,
	// but that API can also lead to "limit exceeded" errors, even if the scope does actually exist.
	scope, err := client.Secrets.ListSecrets(ctx, secrets.ListSecretsRequest{Scope: new(secretScopeName)})
	if err != nil && !apierrs.Is(err, apierrs.ErrResourceDoesNotExist) {
		return "", fmt.Errorf("failed to check if secret scope %s exists: %w", secretScopeName, err)
	}

	if scope != nil && err == nil {
		return secretScopeName, nil
	}

	_, err = client.Secrets.CreateScope(ctx, secrets.CreateScopeRequest{
		Scope: new(secretScopeName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create secrets scope: %w", err)
	}
	return secretScopeName, nil
}

func GetSecret(ctx context.Context, client *sdkclients.Clients, scope, key string) ([]byte, error) {
	resp, err := client.Secrets.GetSecret(ctx, secrets.GetSecretRequest{
		Scope: new(scope),
		Key:   new(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s from scope %s: %w", key, scope, err)
	}
	// The API returns the value base64-encoded; sdk-go decodes it.
	return resp.Value, nil
}

func putSecret(ctx context.Context, client *sdkclients.Clients, scope, key, value string) error {
	_, err := client.Secrets.PutSecret(ctx, secrets.PutSecretRequest{
		Scope: new(scope),
		Key:   new(key),
		Value: &secrets.PutSecretRequest_Value_StringValue{StringValue: value},
	})
	if err != nil {
		return fmt.Errorf("failed to store secret %s in scope %s: %w", key, scope, err)
	}
	return nil
}
