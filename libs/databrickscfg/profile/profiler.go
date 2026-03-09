package profile

import (
	"context"

	"github.com/databricks/cli/libs/databrickscfg"
)

type ProfileMatchFunction func(Profile) bool

func MatchWorkspaceProfiles(p Profile) bool {
	// Match workspace profiles: regular workspace profiles (no account ID)
	// or unified hosts with workspace ID
	return (p.AccountID == "" && !p.IsUnifiedHost) ||
		(p.IsUnifiedHost && p.WorkspaceID != "")
}

func MatchAccountProfiles(p Profile) bool {
	// Match account profiles: regular account profiles (with account ID)
	// or unified hosts with account ID but no workspace ID
	return (p.Host != "" && p.AccountID != "" && !p.IsUnifiedHost) ||
		(p.IsUnifiedHost && p.AccountID != "" && p.WorkspaceID == "")
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
	target := databrickscfg.NormalizeHost(host)
	return func(p Profile) bool {
		return p.Host != "" && databrickscfg.NormalizeHost(p.Host) == target
	}
}

// WithHostAndAccountID returns a ProfileMatchFunction that matches profiles
// by both canonical host and account ID.
func WithHostAndAccountID(host, accountID string) ProfileMatchFunction {
	target := databrickscfg.NormalizeHost(host)
	return func(p Profile) bool {
		return p.Host != "" && databrickscfg.NormalizeHost(p.Host) == target && p.AccountID == accountID
	}
}

type Profiler interface {
	LoadProfiles(context.Context, ProfileMatchFunction) (Profiles, error)
	GetPath(context.Context) (string, error)
}

var DefaultProfiler = FileProfilerImpl{}
