package profile

import (
	"context"
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

func WithName(name string) ProfileMatchFunction {
	return func(p Profile) bool {
		return p.Name == name
	}
}

type Profiler interface {
	LoadProfiles(context.Context, ProfileMatchFunction) (Profiles, error)
	GetPath(context.Context) (string, error)
}

var DefaultProfiler = FileProfilerImpl{}
