package profile

import (
	"context"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/config"
)

type ProfileMatchFunction func(Profile) bool

func MatchWorkspaceProfiles(p Profile) bool {
	// Workspace profile: has workspace_id (covers both classic and SPOG profiles),
	// or is a regular workspace host (no account_id).
	// workspace_id = "none" is a sentinel for "skip workspace", so it does NOT count.
	return (p.WorkspaceID != "" && p.WorkspaceID != auth.WorkspaceIDNone) || p.AccountID == ""
}

func MatchAccountProfiles(p Profile) bool {
	// Account profile: has host and account_id but no workspace_id.
	// workspace_id = "none" is a sentinel for account-level access, treated as empty.
	// This covers classic accounts.* profiles, legacy unified-host account profiles,
	// and new SPOG account profiles.
	return p.Host != "" && p.AccountID != "" && (p.WorkspaceID == "" || p.WorkspaceID == auth.WorkspaceIDNone)
}

func MatchAllProfiles(p Profile) bool {
	return true
}

// MatchProfileNames returns a match function that matches profiles by name.
func MatchProfileNames(names ...string) ProfileMatchFunction {
	nameSet := make(map[string]struct{}, len(names))
	for _, n := range names {
		nameSet[n] = struct{}{}
	}
	return func(p Profile) bool {
		_, ok := nameSet[p.Name]
		return ok
	}
}

func WithName(name string) ProfileMatchFunction {
	return func(p Profile) bool {
		return p.Name == name
	}
}

// WithHost returns a ProfileMatchFunction that matches profiles whose
// canonical host equals the given host.
func WithHost(host string) ProfileMatchFunction {
	target := canonicalizeHost(host)
	return func(p Profile) bool {
		return p.Host != "" && canonicalizeHost(p.Host) == target
	}
}

// WithHostAndAccountID returns a ProfileMatchFunction that matches profiles
// by both canonical host and account ID.
func WithHostAndAccountID(host, accountID string) ProfileMatchFunction {
	target := canonicalizeHost(host)
	return func(p Profile) bool {
		return p.Host != "" && canonicalizeHost(p.Host) == target && p.AccountID == accountID
	}
}

// WithHostAccountIDAndWorkspaceID returns a ProfileMatchFunction that matches
// profiles by canonical host, account ID, and workspace ID. This is used for
// SPOG workspace profiles where multiple workspaces share the same host and
// account ID.
func WithHostAccountIDAndWorkspaceID(host, accountID, workspaceID string) ProfileMatchFunction {
	target := canonicalizeHost(host)
	return func(p Profile) bool {
		return p.Host != "" && canonicalizeHost(p.Host) == target && p.AccountID == accountID && p.WorkspaceID == workspaceID
	}
}

// canonicalizeHost normalizes a host using the SDK's canonical host logic.
func canonicalizeHost(host string) string {
	return (&config.Config{Host: host}).CanonicalHostName()
}

type Profiler interface {
	LoadProfiles(context.Context, ProfileMatchFunction) (Profiles, error)
	GetPath(context.Context) (string, error)
}

var DefaultProfiler = FileProfilerImpl{}
