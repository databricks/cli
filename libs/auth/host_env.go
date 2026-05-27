package auth

import "os"

// SPOG URLs from the Databricks UI carry the workspace ID as a ?o= query
// parameter and the account ID as ?a=, e.g.
// https://acme.databricks.net/?o=12345. The SDK strips path and query from
// Host in fixHostIfNeeded without extracting these IDs, so pasting such a
// URL into DATABRICKS_HOST drops the workspace identifier and API calls hit
// the SPOG without an X-Databricks-Org-Id header, which the server answers
// with HTML (a login page) instead of JSON.
const (
	envHost        = "DATABRICKS_HOST"
	envWorkspaceID = "DATABRICKS_WORKSPACE_ID"
	envAccountID   = "DATABRICKS_ACCOUNT_ID"
)

// NormalizeDatabricksHostEnv extracts ?o=/?workspace_id= and ?a=/?account_id=
// from DATABRICKS_HOST and promotes them to DATABRICKS_WORKSPACE_ID and
// DATABRICKS_ACCOUNT_ID respectively, then rewrites DATABRICKS_HOST without
// the query string. Existing values of the destination env vars are never
// overwritten. Safe to call when DATABRICKS_HOST is unset or has no query.
func NormalizeDatabricksHostEnv() {
	host, ok := os.LookupEnv(envHost)
	if !ok || host == "" {
		return
	}
	params := ExtractHostQueryParams(host)
	if params.Host == host {
		return
	}
	os.Setenv(envHost, params.Host)
	if params.WorkspaceID != "" {
		if _, set := os.LookupEnv(envWorkspaceID); !set {
			os.Setenv(envWorkspaceID, params.WorkspaceID)
		}
	}
	if params.AccountID != "" {
		if _, set := os.LookupEnv(envAccountID); !set {
			os.Setenv(envAccountID, params.AccountID)
		}
	}
}
