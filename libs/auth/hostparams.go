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

	// WorkspaceID extracted from ?o= or ?workspace_id=.
	// Empty if not present or not numeric.
	WorkspaceID string

	// AccountID extracted from ?a= or ?account_id=.
	// Empty if not present.
	AccountID string
}

// ExtractHostQueryParams parses recognized query parameters from a host URL.
// Recognized parameters: o (workspace_id), workspace_id, a (account_id), account_id.
// Workspace IDs must be numeric; non-numeric values are ignored.
// The returned Host has all query parameters and fragments stripped.
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
