package auth

import (
	"net/url"
	"strconv"
	"strings"
)

// HostParams holds query parameters extracted from a host URL.
type HostParams struct {
	// Host is the URL with query parameters stripped.
	Host string

	// WorkspaceID extracted from ?o=, ?w=, or ?workspace_id=.
	// Empty if not present. ?o= and ?workspace_id= are legacy spellings that
	// remain numeric-only; ?w= is the new spelling and is passed through
	// unchanged so non-numeric connection-style identifiers reach the server.
	WorkspaceID string

	// AccountID extracted from ?a= or ?account_id=.
	// Empty if not present.
	AccountID string
}

// ExtractHostQueryParams parses recognized query parameters from a host URL.
// Recognized parameters: o (workspace_id), w (workspace_id), workspace_id,
// a (account_id), account_id. The "w" spelling matches the new
// X-Databricks-Workspace-Id routing header and accepts any non-empty value
// (including non-numeric connection-style identifiers). The legacy "o" and
// "workspace_id" spellings remain numeric-only — they predate the broader
// identifier shapes and historical URLs carrying those forms are always
// numeric. When more than one spelling is present, "o" wins to preserve the
// meaning of existing URLs. The returned Host has all query parameters and
// fragments stripped.
func ExtractHostQueryParams(host string) HostParams {
	u, err := url.Parse(host)
	if err != nil || u.RawQuery == "" {
		return HostParams{Host: host}
	}

	q := u.Query()

	var workspaceID string
	if v := q.Get("o"); v != "" {
		if _, err := strconv.ParseInt(v, 10, 64); err == nil {
			workspaceID = v
		}
	} else if v := q.Get("w"); v != "" {
		workspaceID = v
	} else if v := q.Get("workspace_id"); v != "" {
		if _, err := strconv.ParseInt(v, 10, 64); err == nil {
			workspaceID = v
		}
	}

	var accountID string
	if v := q.Get("a"); v != "" {
		accountID = v
	} else if v := q.Get("account_id"); v != "" {
		accountID = v
	}

	// Strip query params from host.
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimSuffix(u.Path, "/")

	return HostParams{
		Host:        u.String(),
		WorkspaceID: workspaceID,
		AccountID:   accountID,
	}
}
