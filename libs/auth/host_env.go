package auth

import (
	"context"

	"github.com/databricks/cli/libs/env"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

// SPOG URLs from the Databricks UI carry the workspace ID as a ?o= query
// parameter and the account ID as ?a=, e.g.
// https://acme.databricks.net/?o=12345. The SDK strips path and query from
// Host in fixHostIfNeeded without extracting these IDs, so a DATABRICKS_HOST
// env var with such a URL drops the workspace identifier and API calls hit
// the SPOG without an X-Databricks-Org-Id header, which the server answers
// with HTML (a login page) instead of JSON.
//
// TODO: stopgap. The matching SDK fix is databricks/databricks-sdk-go#1699,
// which handles ?o=/?a= directly in fixHostIfNeeded. Delete this helper on
// the next SDK bump that includes that change.

// NormalizeDatabricksConfigFromEnv promotes ?o=/?workspace_id= and
// ?a=/?account_id= query parameters from the DATABRICKS_HOST env var into
// the matching fields on cfg, and sets cfg.Host to the stripped URL. It
// does not mutate process env, so the effect is scoped to the SDK config
// built from this cfg (and any subprocess env derived from it via
// auth.Env).
//
// Only fills in empty fields. If cfg.Host is already set, the query
// params aren't promoted at all (an explicit host takes priority). If a
// dedicated env var (DATABRICKS_WORKSPACE_ID, DATABRICKS_ACCOUNT_ID) is
// set, that more explicit signal wins over the query param.
func NormalizeDatabricksConfigFromEnv(ctx context.Context, cfg *sdkconfig.Config) {
	if cfg.Host != "" {
		return
	}
	host, ok := env.Lookup(ctx, "DATABRICKS_HOST")
	if !ok || host == "" {
		return
	}
	params := ExtractHostQueryParams(host)
	if params.Host == host {
		return
	}
	cfg.Host = params.Host
	if cfg.WorkspaceID == "" && params.WorkspaceID != "" && env.Get(ctx, "DATABRICKS_WORKSPACE_ID") == "" {
		cfg.WorkspaceID = params.WorkspaceID
	}
	if cfg.AccountID == "" && params.AccountID != "" && env.Get(ctx, "DATABRICKS_ACCOUNT_ID") == "" {
		cfg.AccountID = params.AccountID
	}
}
