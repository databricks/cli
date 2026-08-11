package aircmd

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// Discovered Docker credentials are stored in a per-Databricks-user secret. The
// scope is per-user (creator-only ACL) so one user's registry PAT is never
// readable by other workspace members. The key is "<registry-host>-<username>"
// plus a suffix, so auto-managed keys are distinguishable, never overwrite a
// hand-curated secret, and don't collide across registries.
const (
	dockerCredsScopePrefix = "docker-credentials"
	localManagedKeySuffix  = "-local"
)

// errSecretScopeQuota signals the workspace has hit its secret-scope limit. It
// is surfaced to the user rather than swallowed, since it only resolves by
// freeing a scope, not by retrying or falling back to the public-image path.
var errSecretScopeQuota = errors.New("workspace has reached the maximum number of secret scopes; delete an unused scope and retry")

// encodeDockerCredentials base64-encodes "username:password", the form the
// registration backend decodes.
func encodeDockerCredentials(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// isScopeQuotaError reports whether err is a secret-scope quota rejection.
func isScopeQuotaError(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.ErrorCode == "RESOURCE_LIMIT_EXCEEDED"
}

// ensureSecretScope creates scope if it does not already exist, using the API
// default ACL (creator-only MANAGE). It must not grant workspace-wide access: a
// Docker-credential scope readable by every member would leak the user's PAT.
func ensureSecretScope(ctx context.Context, w *databricks.WorkspaceClient, scope string) error {
	scopes, err := w.Secrets.ListScopesAll(ctx)
	if err == nil {
		for _, s := range scopes {
			if s.Name == scope {
				return nil
			}
		}
	} else {
		// A user without LIST permission can still create their own scope, so
		// treat a list failure as "unknown" and proceed to create.
		log.Debugf(ctx, "could not list secret scopes: %v", err)
	}

	err = w.Secrets.CreateScope(ctx, workspace.CreateScope{Scope: scope})
	switch {
	case err == nil:
		return nil
	case errors.Is(err, apierr.ErrResourceAlreadyExists):
		return nil
	case isScopeQuotaError(err):
		return errSecretScopeQuota
	case errors.Is(err, apierr.ErrPermissionDenied):
		return fmt.Errorf("creating secret scope %q was denied (%w). Ask a workspace admin for permission to create secret scopes", scope, err)
	default:
		return fmt.Errorf("creating secret scope %q failed: %w", scope, err)
	}
}

// storeDockerCredentials stores registry credentials in the per-user secret
// scope and returns the (scope, key) reference for registration. A storage
// failure is returned rather than swallowed: registration continues without
// credentials (a public image still succeeds), but the caller reports this as the
// cause if the registry then rejects anonymous access. The caller resolves the
// credentials so the local Docker config is read only once, and passes
// normalizedImageURL so the key is namespaced by registry host.
func storeDockerCredentials(ctx context.Context, w *databricks.WorkspaceClient, normalizedImageURL, username, password string) (scope, key string, err error) {
	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		return "", "", fmt.Errorf("could not resolve the current Databricks user: %w", err)
	}

	// Namespace the key by registry host so the same username on two registries
	// (e.g. docker.io and nvcr.io) doesn't collide on one secret.
	host, _, _ := strings.Cut(normalizedImageURL, "/")
	scope = fmt.Sprintf("%s-%s", dockerCredsScopePrefix, me.UserName)
	key = fmt.Sprintf("%s-%s%s", host, username, localManagedKeySuffix)

	if err := ensureSecretScope(ctx, w, scope); err != nil {
		return "", "", err
	}

	if err := w.Secrets.PutSecret(ctx, workspace.PutSecret{
		Scope:       scope,
		Key:         key,
		StringValue: encodeDockerCredentials(username, password),
	}); err != nil {
		if isScopeQuotaError(err) {
			return "", "", errSecretScopeQuota
		}
		return "", "", fmt.Errorf("storing credentials in secret %s/%s failed: %w", scope, key, err)
	}

	return scope, key, nil
}
