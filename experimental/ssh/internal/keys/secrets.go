package keys

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// CreateKeysSecretScope creates or retrieves the secret scope for SSH keys.
// sessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
// serverless only selects the recovery advice given when an existing scope is refused.
func CreateKeysSecretScope(ctx context.Context, client *databricks.WorkspaceClient, sessionID string, serverless bool) (string, error) {
	me, err := client.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	secretScopeName := fmt.Sprintf("%s-%s-ssh-tunnel-keys", me.UserName, sessionID)

	// Do not create the scope if it already exists.
	// We can instead filter out "resource already exists" errors from CreateScope,
	// but that API can also lead to "limit exceeded" errors, even if the scope does actually exist.
	scope, err := client.Secrets.ListSecretsByScope(ctx, secretScopeName)
	if err != nil && !errors.Is(err, databricks.ErrResourceDoesNotExist) {
		return "", fmt.Errorf("failed to check if secret scope %s exists: %w", secretScopeName, err)
	}

	if scope != nil && err == nil {
		if err := checkScopeBackend(ctx, client, secretScopeName, serverless); err != nil {
			return "", err
		}
		if err := checkScopeOwnership(ctx, client, secretScopeName, me.UserName, serverless); err != nil {
			return "", err
		}
		return secretScopeName, nil
	}

	err = client.Secrets.CreateScope(ctx, workspace.CreateScope{
		Scope: secretScopeName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create secrets scope: %w", err)
	}
	return secretScopeName, nil
}

// checkScopeBackend returns an error unless the existing scope is Databricks-backed. The ACL
// of a scope backed by Azure Key Vault governs only access through Databricks: whoever
// controls the vault can read and replace the keys regardless of it.
func checkScopeBackend(ctx context.Context, client *databricks.WorkspaceClient, scope string, serverless bool) error {
	// The Secrets API has no call that returns a single scope, so find it in the list.
	scopes, err := client.Secrets.ListScopesAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to list secret scopes: %w", err)
	}
	for _, s := range scopes {
		if s.Name != scope {
			continue
		}
		if s.BackendType != workspace.ScopeBackendTypeDatabricks {
			return scopeRefusedError(scope, fmt.Sprintf("is backed by %s (the CLI keeps SSH keys only in a %s-backed scope)", s.BackendType, workspace.ScopeBackendTypeDatabricks), serverless)
		}
		return nil
	}
	return fmt.Errorf("secret scope %s is missing from the list of secret scopes", scope)
}

// checkScopeOwnership returns an error unless userName is the only principal with access
// to an existing keys scope: its ACL must have exactly one entry, MANAGE for userName, as the
// service grants to the creator of a scope the CLI creates (CreateScope sets no
// initial_manage_principal). This keeps other principals from reading or replacing keys while
// the scope is in use. It does not prove who created the scope or wrote its secrets: another
// principal can write keys, grant userName MANAGE and then remove its own entry, and the
// client and server still use the keys the scope holds.
func checkScopeOwnership(ctx context.Context, client *databricks.WorkspaceClient, scope, userName string, serverless bool) error {
	acls, err := client.Secrets.ListAclsAll(ctx, workspace.ListAclsRequest{Scope: scope})
	// Listing ACLs requires MANAGE, so a denial means the user does not own the scope.
	if errors.Is(err, databricks.ErrPermissionDenied) {
		return scopeNotOwnedError(scope, userName, serverless)
	}
	if err != nil {
		return fmt.Errorf("failed to list ACLs of secret scope %s: %w", scope, err)
	}
	if len(acls) != 1 || acls[0].Principal != userName || acls[0].Permission != workspace.AclPermissionManage {
		return scopeNotOwnedError(scope, userName, serverless)
	}
	return nil
}

// scopeNotOwnedError advises deleting the scope and connecting again once no server for the
// session is running. The CLI reuses a running server, which keeps authorizing the client key
// it read at startup, while the recreated scope gets a new client key pair and holds no host
// key to pin, so connecting before that server shuts down fails.
func scopeNotOwnedError(scope, userName string, serverless bool) error {
	return scopeRefusedError(scope, fmt.Sprintf("is not exclusively owned by %s (the scope must grant MANAGE to %s and no access to other principals)", userName, userName), serverless)
}

// scopeRefusedError reports that an existing keys scope cannot be reused because of problem,
// with the recovery advice described at scopeNotOwnedError.
func scopeRefusedError(scope, problem string, serverless bool) error {
	msg := fmt.Sprintf("secret scope %s already exists and %s. "+
		"Delete the scope with `databricks secrets delete-scope %s` (this needs MANAGE permission on the scope; otherwise ask a workspace admin) "+
		"and connect again once no SSH server for this session is running: a running server keeps the keys of the deleted scope "+
		"and shuts down --shutdown-delay after its last client disconnects", scope, problem, scope)
	// The serverless session ID is the connection name; a dedicated cluster's is its ID.
	if serverless {
		msg += ". You can instead connect with a different --name"
	}
	return errors.New(msg)
}

func GetSecret(ctx context.Context, client *databricks.WorkspaceClient, scope, key string) ([]byte, error) {
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
