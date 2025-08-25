package ssh

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

func createSecretsScope(ctx context.Context, client *databricks.WorkspaceClient, clusterID string) (string, error) {
	me, err := client.CurrentUser.Me(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	secretsScope := fmt.Sprintf("%s-%s-ssh-tunnel-keys", me.UserName, clusterID)
	err = client.Secrets.CreateScope(ctx, workspace.CreateScope{
		Scope: secretsScope,
	})
	if err != nil && !errors.Is(err, databricks.ErrResourceAlreadyExists) {
		return "", fmt.Errorf("failed to create secrets scope: %w", err)
	}
	return secretsScope, nil
}

func getSecret(ctx context.Context, client *databricks.WorkspaceClient, scope, key string) ([]byte, error) {
	resp, err := client.Secrets.GetSecret(ctx, workspace.GetSecretRequest{
		Scope: scope,
		Key:   key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s from scope %s: %w", key, scope, err)
	}

	value, err := base64.StdEncoding.DecodeString(resp.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret key from base64: %w", err)
	}
	return value, nil
}

func putSecret(ctx context.Context, client *databricks.WorkspaceClient, scope, key, value string) error {
	err := client.Secrets.PutSecret(ctx, workspace.PutSecret{
		Scope:       scope,
		Key:         key,
		StringValue: value,
	})
	if err != nil {
		return fmt.Errorf("failed to store secret %s in scope %s: %w", key, scope, err)
	}
	return nil
}

func putSecretInScope(ctx context.Context, client *databricks.WorkspaceClient, clusterID, key, value string) (string, error) {
	scopeName, err := createSecretsScope(ctx, client, clusterID)
	if err != nil {
		return "", err
	}
	err = putSecret(ctx, client, scopeName, key, value)
	if err != nil {
		return "", err
	}
	return scopeName, nil
}
